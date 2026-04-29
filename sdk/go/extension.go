package aghsdk

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"
	"sync"
)

// ExtensionContext is passed to custom service handlers.
type ExtensionContext struct {
	Request   JSONRPCRequestEnvelope
	RequestID json.RawMessage
	Host      *HostAPI
	Session   ExtensionSession
	Logf      func(format string, args ...any)
}

// ExtensionHandler handles one custom AGH -> extension service request.
type ExtensionHandler func(context.Context, ExtensionContext, json.RawMessage) (any, error)

// ToolRequest is passed to a typed tool handler.
type ToolRequest[TInput any] struct {
	Input    TInput
	RawInput json.RawMessage
	Context  ExtensionContext
	Host     *HostAPI
	Session  ExtensionSession
	ToolID   ToolID
	Handler  string
}

// ToolHandlerFunc handles one typed tool request.
type ToolHandlerFunc[TInput any] func(context.Context, ToolRequest[TInput]) (ToolResult, error)

type registeredTool struct {
	descriptor           ExtensionToolRuntimeDescriptor
	handler              func(context.Context, rawToolRequest) (ToolResult, error)
	sensitiveInputFields []string
}

type rawToolRequest struct {
	input   json.RawMessage
	context ExtensionContext
	toolID  ToolID
	handler string
}

// Extension is a subprocess-hosted AGH extension runtime.
type Extension struct {
	definition ExtensionDefinition
	transport  Transport
	stderr     io.Writer
	sdkVersion string
	host       *HostAPI

	mu                 sync.RWMutex
	handlers           map[string]ExtensionHandler
	toolHandlers       map[string]registeredTool
	readyCallbacks     []func(context.Context, *HostAPI, ExtensionSession) error
	initialized        bool
	shutdownStarted    bool
	shutdownDeadlineMS int64
	session            *ExtensionSession
}

// Option configures an Extension.
type Option func(*Extension)

// WithTransport sets a custom JSON-RPC transport.
func WithTransport(transport Transport) Option {
	return func(extension *Extension) {
		if transport != nil {
			extension.transport = transport
		}
	}
}

// WithStdio configures stdio input and output streams.
func WithStdio(input io.Reader, output io.Writer) Option {
	return func(extension *Extension) {
		extension.transport = NewStdioTransport(StdioTransportOptions{Input: input, Output: output})
	}
}

// WithStderr sets the log sink used by ExtensionContext.Logf.
func WithStderr(stderr io.Writer) Option {
	return func(extension *Extension) {
		if stderr != nil {
			extension.stderr = stderr
		}
	}
}

// WithSDKVersion overrides the advertised SDK version.
func WithSDKVersion(version string) Option {
	return func(extension *Extension) {
		if strings.TrimSpace(version) != "" {
			extension.sdkVersion = strings.TrimSpace(version)
		}
	}
}

// NewExtension creates a public Go extension runtime.
func NewExtension(definition ExtensionDefinition, options ...Option) *Extension {
	extension := &Extension{
		definition:   definition,
		transport:    NewStdioTransport(StdioTransportOptions{}),
		stderr:       os.Stderr,
		sdkVersion:   SDKVersion,
		handlers:     make(map[string]ExtensionHandler),
		toolHandlers: make(map[string]registeredTool),
	}
	extension.host = newHostAPI(extension.transport, extension.ready)
	for _, option := range options {
		option(extension)
	}
	extension.host = newHostAPI(extension.transport, extension.ready)
	extension.bindBuiltins()
	return extension
}

// Handle registers one custom AGH -> extension service method.
func (e *Extension) Handle(method string, handler ExtensionHandler) error {
	if e == nil {
		return NewInternalError("extension is required")
	}
	cleanMethod := strings.TrimSpace(method)
	if cleanMethod == "" {
		return NewInvalidParamsError("method is required", nil)
	}
	if cleanMethod == initializeMethod {
		return NewInvalidParamsError("initialize is reserved by the SDK", nil)
	}
	if e.hasToolHandlers() && isToolProviderMethod(cleanMethod) {
		return NewInvalidParamsError(cleanMethod+" is reserved by Tool", nil)
	}
	if handler == nil {
		return NewInvalidParamsError("handler is required", nil)
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	e.handlers[cleanMethod] = handler
	e.transport.Handle(cleanMethod, e.dispatch)
	return nil
}

// Tool registers a raw JSON tool handler on the extension.
func (e *Extension) Tool(
	handler string,
	options ToolOptions,
	fn ToolHandlerFunc[json.RawMessage],
) error {
	return Tool[json.RawMessage](e, handler, options, fn)
}

// Tool registers a typed Go function as an extension-host tool handler.
func Tool[TInput any](
	extension *Extension,
	handler string,
	options ToolOptions,
	fn ToolHandlerFunc[TInput],
) error {
	if extension == nil {
		return NewInternalError("extension is required")
	}
	if fn == nil {
		return NewInvalidParamsError("tool handler function is required", nil)
	}
	return extension.registerTool(handler, options, func(ctx context.Context, req rawToolRequest) (ToolResult, error) {
		var input TInput
		rawInput := req.input
		if len(rawInput) == 0 {
			rawInput = json.RawMessage(`{}`)
		}
		if err := json.Unmarshal(rawInput, &input); err != nil {
			return ToolResult{}, NewInvalidParamsError("tool input does not match handler type", map[string]any{
				"handler": req.handler,
				"error":   err.Error(),
			})
		}
		return fn(ctx, ToolRequest[TInput]{
			Input:    input,
			RawInput: cloneRawMessage(rawInput),
			Context:  req.context,
			Host:     req.context.Host,
			Session:  req.context.Session,
			ToolID:   req.toolID,
			Handler:  req.handler,
		})
	})
}

// OnReady registers a callback that runs after initialize succeeds.
func (e *Extension) OnReady(callback func(context.Context, *HostAPI, ExtensionSession) error) {
	if e == nil || callback == nil {
		return
	}
	e.mu.Lock()
	e.readyCallbacks = append(e.readyCallbacks, callback)
	session := e.session
	initialized := e.initialized && session != nil
	host := e.host
	e.mu.Unlock()
	if initialized {
		go func() {
			e.runReadyCallback(context.Background(), callback, host, session)
		}()
	}
}

// Run starts the JSON-RPC transport and blocks until it closes.
func (e *Extension) Run(ctx context.Context) error {
	if e == nil {
		return NewInternalError("extension is required")
	}
	if err := e.definition.validate(); err != nil {
		return err
	}
	return e.transport.Run(ctx)
}

// GetImplementedMethods returns the sorted method list advertised during initialize.
func (e *Extension) GetImplementedMethods() []string {
	if e == nil {
		return nil
	}
	e.mu.RLock()
	defer e.mu.RUnlock()
	methods := map[string]struct{}{
		healthCheckMethod: {},
		shutdownMethod:    {},
	}
	for method := range e.handlers {
		methods[method] = struct{}{}
	}
	if len(e.toolHandlers) > 0 {
		methods[ExtensionServiceMethodProvideTools] = struct{}{}
		methods[ExtensionServiceMethodToolsCall] = struct{}{}
	}
	out := make([]string, 0, len(methods))
	for method := range methods {
		out = append(out, method)
	}
	slices.Sort(out)
	return out
}

// GetToolDescriptors returns runtime descriptors registered by Tool.
func (e *Extension) GetToolDescriptors() []ExtensionToolRuntimeDescriptor {
	if e == nil {
		return nil
	}
	e.mu.RLock()
	defer e.mu.RUnlock()
	descriptors := make([]ExtensionToolRuntimeDescriptor, 0, len(e.toolHandlers))
	for _, tool := range e.toolHandlers {
		descriptor := tool.descriptor
		descriptor.Capabilities = slices.Clone(descriptor.Capabilities)
		descriptors = append(descriptors, descriptor)
	}
	slices.SortFunc(descriptors, func(a, b ExtensionToolRuntimeDescriptor) int {
		return strings.Compare(a.Handler, b.Handler)
	})
	return descriptors
}

func (e *Extension) registerTool(
	handler string,
	options ToolOptions,
	fn func(context.Context, rawToolRequest) (ToolResult, error),
) error {
	cleanHandler := strings.TrimSpace(handler)
	if cleanHandler == "" {
		return NewInvalidParamsError("tool handler is required", nil)
	}
	inputSchema, err := normalizeSchema(options.InputSchema, "input_schema", true)
	if err != nil {
		return err
	}
	outputSchema, err := normalizeSchema(options.OutputSchema, "output_schema", false)
	if err != nil {
		return err
	}
	toolID := options.ID
	if toolID == "" {
		toolID, err = canonicalExtensionToolID(e.definition.Name, cleanHandler)
		if err != nil {
			return err
		}
	}
	if err := toolID.Validate(); err != nil {
		return err
	}
	inputDigest, err := SchemaDigest(inputSchema)
	if err != nil {
		return err
	}
	outputDigest := ""
	if len(outputSchema) > 0 {
		outputDigest, err = SchemaDigest(outputSchema)
		if err != nil {
			return err
		}
	}
	risk := options.Risk
	if risk == "" {
		if options.ReadOnly {
			risk = RiskRead
		} else {
			risk = RiskMutating
		}
	}
	descriptor := ExtensionToolRuntimeDescriptor{
		ID:                 toolID,
		Handler:            cleanHandler,
		InputSchemaDigest:  inputDigest,
		OutputSchemaDigest: outputDigest,
		ReadOnly:           options.ReadOnly,
		Risk:               risk,
		Capabilities:       normalizeStrings(options.Capabilities),
	}

	e.mu.Lock()
	defer e.mu.Unlock()
	if err := e.ensureToolRegistrationAvailableLocked(cleanHandler, toolID); err != nil {
		return err
	}
	e.toolHandlers[cleanHandler] = registeredTool{
		descriptor:           descriptor,
		handler:              fn,
		sensitiveInputFields: normalizeStrings(options.SensitiveInputFields),
	}
	e.definition.Capabilities.Provides = normalizeStrings(
		append(e.definition.Capabilities.Provides, CapabilityToolProvider),
	)
	e.transport.Handle(ExtensionServiceMethodProvideTools, e.dispatch)
	e.transport.Handle(ExtensionServiceMethodToolsCall, e.dispatch)
	return nil
}

func (e *Extension) ensureToolRegistrationAvailableLocked(handler string, toolID ToolID) error {
	if _, ok := e.toolHandlers[handler]; ok {
		return NewInvalidParamsError("tool handler already registered", map[string]any{"handler": handler})
	}
	for existingHandler, existingTool := range e.toolHandlers {
		if existingTool.descriptor.ID == toolID {
			return NewInvalidParamsError("tool id already registered", map[string]any{
				"tool_id":          toolID,
				"existing_handler": existingHandler,
				"handler":          handler,
			})
		}
	}
	if _, ok := e.handlers[ExtensionServiceMethodProvideTools]; ok {
		return NewInvalidParamsError("provide_tools is reserved by Tool", nil)
	}
	if _, ok := e.handlers[ExtensionServiceMethodToolsCall]; ok {
		return NewInvalidParamsError("tools/call is reserved by Tool", nil)
	}
	return nil
}

func (e *Extension) bindBuiltins() {
	e.transport.Handle(initializeMethod, e.dispatch)
	e.transport.Handle(healthCheckMethod, e.dispatch)
	e.transport.Handle(shutdownMethod, e.dispatch)
}

func (e *Extension) dispatch(
	ctx context.Context,
	params json.RawMessage,
	request JSONRPCRequestEnvelope,
) (any, error) {
	switch request.Method {
	case initializeMethod:
		return e.handleInitialize(params)
	case healthCheckMethod:
		if err := e.ensureReady(); err != nil {
			return nil, err
		}
		return e.handleHealthCheck(ctx, request, params)
	case shutdownMethod:
		if err := e.ensureReady(); err != nil {
			return nil, err
		}
		return e.handleShutdown(ctx, request, params)
	case ExtensionServiceMethodProvideTools:
		if err := e.ensureReady(); err != nil {
			return nil, err
		}
		return ExtensionProvideToolsResponse{Tools: e.GetToolDescriptors()}, nil
	case ExtensionServiceMethodToolsCall:
		if err := e.ensureReady(); err != nil {
			return nil, err
		}
		return e.handleToolCall(ctx, request, params)
	default:
		if err := e.ensureReady(); err != nil {
			return nil, err
		}
		return e.handleUserMethod(ctx, request, params)
	}
}

func (e *Extension) handleInitialize(params json.RawMessage) (InitializeResponse, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.initialized {
		return InitializeResponse{}, NewInvalidParamsError("initialize may only be called once", nil)
	}
	if err := e.definition.validate(); err != nil {
		return InitializeResponse{}, err
	}
	var request InitializeRequest
	if err := json.Unmarshal(params, &request); err != nil {
		return InitializeResponse{}, NewInvalidParamsError(
			"initialize params must be an object",
			map[string]any{"error": err.Error()},
		)
	}
	if err := validateInitializeRequest(request); err != nil {
		return InitializeResponse{}, err
	}
	if request.ProtocolVersion != ProtocolVersion ||
		!slices.Contains(request.SupportedProtocolVersions, ProtocolVersion) {
		return InitializeResponse{}, NewInvalidParamsError("unsupported protocol version", map[string]any{
			"protocol_version": request.ProtocolVersion,
		})
	}
	requestedProvides := normalizeStrings(e.definition.Capabilities.Provides)
	requestedActions := normalizeHostMethods(e.definition.Actions.Requires)
	requestedSecurity := normalizeStrings(e.definition.Security.Capabilities)
	grantedActions := hostMethodsToStrings(request.Capabilities.GrantedActions)
	if err := ensureSubset("provides", requestedProvides, request.Capabilities.Provides); err != nil {
		return InitializeResponse{}, err
	}
	if err := ensureSubset("actions", hostMethodsToStrings(requestedActions), grantedActions); err != nil {
		return InitializeResponse{}, err
	}
	if err := ensureSubset("security", requestedSecurity, request.Capabilities.GrantedSecurity); err != nil {
		return InitializeResponse{}, err
	}
	implemented := e.implementedMethodsLocked()
	if err := validateProvidedMethodCoverage(requestedProvides, implemented); err != nil {
		return InitializeResponse{}, err
	}
	response := InitializeResponse{
		ProtocolVersion: ProtocolVersion,
		ExtensionInfo: InitializeExtensionInfo{
			Name:       e.definition.Name,
			Version:    e.definition.Version,
			SDKName:    SDKName,
			SDKVersion: e.sdkVersion,
		},
		AcceptedCapabilities: AcceptedCapabilities{
			Provides: requestedProvides,
			Actions:  requestedActions,
			Security: requestedSecurity,
		},
		ImplementedMethods:  implemented,
		SupportedHookEvents: normalizeStrings(e.definition.SupportedHookEvents),
		Supports:            InitializeSupports{HealthCheck: true},
	}
	session := ExtensionSession{
		InitializeRequest:    request,
		InitializeResponse:   response,
		Runtime:              request.Runtime,
		AcceptedCapabilities: response.AcceptedCapabilities,
	}
	e.initialized = true
	e.session = &session
	go e.runReadyCallbacks(context.Background(), &session)
	return response, nil
}

func (e *Extension) handleHealthCheck(
	ctx context.Context,
	request JSONRPCRequestEnvelope,
	params json.RawMessage,
) (HealthCheckResult, error) {
	e.mu.RLock()
	handler := e.handlers[healthCheckMethod]
	e.mu.RUnlock()
	if handler == nil {
		return HealthCheckResult{Healthy: true, Message: "", Details: map[string]json.RawMessage{}}, nil
	}
	result, err := handler(ctx, e.makeContext(request), params)
	if err != nil {
		return HealthCheckResult{}, err
	}
	typed, ok := result.(HealthCheckResult)
	if !ok {
		return HealthCheckResult{}, NewInternalError("health_check handler must return HealthCheckResult")
	}
	return typed, nil
}

func (e *Extension) handleShutdown(
	ctx context.Context,
	request JSONRPCRequestEnvelope,
	params json.RawMessage,
) (ShutdownResponse, error) {
	var shutdown ShutdownRequest
	if err := json.Unmarshal(params, &shutdown); err != nil {
		return ShutdownResponse{}, NewInvalidParamsError(
			"shutdown params must be an object",
			map[string]any{"error": err.Error()},
		)
	}
	if shutdown.DeadlineMS <= 0 {
		return ShutdownResponse{}, NewInvalidParamsError("deadline_ms must be greater than zero", nil)
	}
	e.mu.Lock()
	e.shutdownStarted = true
	e.shutdownDeadlineMS = shutdown.DeadlineMS
	handler := e.handlers[shutdownMethod]
	e.mu.Unlock()
	if handler == nil {
		return ShutdownResponse{Acknowledged: true}, nil
	}
	result, err := handler(ctx, e.makeContext(request), params)
	if err != nil {
		return ShutdownResponse{}, err
	}
	if result == nil {
		return ShutdownResponse{Acknowledged: true}, nil
	}
	typed, ok := result.(ShutdownResponse)
	if !ok {
		return ShutdownResponse{}, NewInternalError("shutdown handler must return ShutdownResponse")
	}
	return typed, nil
}

func (e *Extension) handleToolCall(
	ctx context.Context,
	request JSONRPCRequestEnvelope,
	params json.RawMessage,
) (ExtensionToolCallResponse, error) {
	var call ExtensionToolCallRequest
	if err := json.Unmarshal(params, &call); err != nil {
		return ExtensionToolCallResponse{}, NewInvalidParamsError(
			"tools/call params must be an object",
			map[string]any{"error": err.Error()},
		)
	}
	call.Handler = strings.TrimSpace(call.Handler)
	if call.Handler == "" {
		return ExtensionToolCallResponse{}, NewInvalidParamsError("handler is required", nil)
	}
	if err := call.ToolID.Validate(); err != nil {
		return ExtensionToolCallResponse{}, err
	}
	e.mu.RLock()
	registered, ok := e.toolHandlers[call.Handler]
	e.mu.RUnlock()
	if !ok {
		return ExtensionToolCallResponse{}, NewMethodNotFoundError(call.Handler)
	}
	if registered.descriptor.ID != call.ToolID {
		return ExtensionToolCallResponse{}, NewInvalidParamsError("tool_id does not match handler", map[string]any{
			"expected_tool_id": registered.descriptor.ID,
			"actual_tool_id":   call.ToolID,
			"handler":          call.Handler,
		})
	}
	contextValue := e.makeContext(request)
	result, err := registered.handler(ctx, rawToolRequest{
		input:   cloneRawMessage(call.Input),
		context: contextValue,
		toolID:  call.ToolID,
		handler: call.Handler,
	})
	if err != nil {
		if rpcErr := ensureRPCErrorIfTyped(err); rpcErr != nil {
			return ExtensionToolCallResponse{}, rpcErr
		}
		return ExtensionToolCallResponse{}, toolExecutionError(err, call, registered.sensitiveInputFields)
	}
	return ExtensionToolCallResponse{Result: result}, nil
}

func (e *Extension) handleUserMethod(
	ctx context.Context,
	request JSONRPCRequestEnvelope,
	params json.RawMessage,
) (any, error) {
	e.mu.RLock()
	handler := e.handlers[request.Method]
	e.mu.RUnlock()
	if handler == nil {
		return nil, NewMethodNotFoundError(request.Method)
	}
	return handler(ctx, e.makeContext(request), params)
}

func (e *Extension) makeContext(request JSONRPCRequestEnvelope) ExtensionContext {
	e.mu.RLock()
	session := e.session
	e.mu.RUnlock()
	if session == nil {
		return ExtensionContext{}
	}
	return ExtensionContext{
		Request:   request,
		RequestID: cloneRawMessage(request.ID),
		Host:      e.host,
		Session:   *session,
		Logf: func(format string, args ...any) {
			if e.stderr != nil {
				fmt.Fprintf(e.stderr, format+"\n", args...)
			}
		},
	}
}

func (e *Extension) ensureReady() error {
	e.mu.RLock()
	defer e.mu.RUnlock()
	if !e.initialized || e.session == nil {
		return NewNotInitializedError()
	}
	if e.shutdownStarted {
		return NewShutdownInProgressError(e.shutdownDeadlineMS)
	}
	return nil
}

func (e *Extension) ready() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.initialized && e.session != nil && !e.shutdownStarted
}

func (e *Extension) hasToolHandlers() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return len(e.toolHandlers) > 0
}

func (e *Extension) implementedMethodsLocked() []string {
	methods := map[string]struct{}{
		healthCheckMethod: {},
		shutdownMethod:    {},
	}
	for method := range e.handlers {
		methods[method] = struct{}{}
	}
	if len(e.toolHandlers) > 0 {
		methods[ExtensionServiceMethodProvideTools] = struct{}{}
		methods[ExtensionServiceMethodToolsCall] = struct{}{}
	}
	out := make([]string, 0, len(methods))
	for method := range methods {
		out = append(out, method)
	}
	slices.Sort(out)
	return out
}

func (e *Extension) runReadyCallbacks(ctx context.Context, session *ExtensionSession) {
	e.mu.RLock()
	callbacks := slices.Clone(e.readyCallbacks)
	host := e.host
	e.mu.RUnlock()
	for _, callback := range callbacks {
		e.runReadyCallback(ctx, callback, host, session)
	}
}

func (e *Extension) runReadyCallback(
	ctx context.Context,
	callback func(context.Context, *HostAPI, ExtensionSession) error,
	host *HostAPI,
	session *ExtensionSession,
) {
	if callback == nil || session == nil {
		return
	}
	if err := callback(ctx, host, *session); err != nil && e.stderr != nil {
		fmt.Fprintf(e.stderr, "onReady callback failed: %v\n", err)
	}
}

func validateInitializeRequest(request InitializeRequest) error {
	if strings.TrimSpace(request.ProtocolVersion) == "" {
		return NewInvalidParamsError("protocol_version is required", nil)
	}
	if strings.TrimSpace(request.SessionNonce) == "" {
		return NewInvalidParamsError("session_nonce is required", nil)
	}
	if len(request.SupportedProtocolVersions) == 0 {
		return NewInvalidParamsError("supported_protocol_versions is required", nil)
	}
	if strings.TrimSpace(request.Extension.Name) == "" || strings.TrimSpace(request.Extension.Version) == "" {
		return NewInvalidParamsError("extension identity is required", nil)
	}
	if request.Runtime.HealthCheckIntervalMS <= 0 {
		return NewInvalidParamsError("health_check_interval_ms must be greater than zero", nil)
	}
	if request.Runtime.HealthCheckTimeoutMS <= 0 {
		return NewInvalidParamsError("health_check_timeout_ms must be greater than zero", nil)
	}
	if request.Runtime.ShutdownTimeoutMS <= 0 {
		return NewInvalidParamsError("shutdown_timeout_ms must be greater than zero", nil)
	}
	if request.Runtime.DefaultHookTimeoutMS <= 0 {
		return NewInvalidParamsError("default_hook_timeout_ms must be greater than zero", nil)
	}
	return nil
}

func normalizeHostMethods(values []HostAPIMethod) []HostAPIMethod {
	normalized := normalizeStrings(hostMethodsToStrings(values))
	out := make([]HostAPIMethod, 0, len(normalized))
	for _, value := range normalized {
		out = append(out, HostAPIMethod(value))
	}
	return out
}

func hostMethodsToStrings(values []HostAPIMethod) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		out = append(out, string(value))
	}
	return out
}

func canonicalExtensionToolID(extensionName string, handler string) (ToolID, error) {
	extensionSegment, err := canonicalIDSegment(extensionName)
	if err != nil {
		return "", err
	}
	handlerSegment, err := canonicalIDSegment(handler)
	if err != nil {
		return "", err
	}
	id := ToolID("ext__" + extensionSegment + "__" + handlerSegment)
	if err := id.Validate(); err != nil {
		return "", err
	}
	return id, nil
}

func canonicalIDSegment(raw string) (string, error) {
	normalized := strings.TrimSpace(strings.ToLower(raw))
	var builder strings.Builder
	lastUnderscore := false
	for _, char := range normalized {
		switch {
		case char >= 'a' && char <= 'z':
			builder.WriteRune(char)
			lastUnderscore = false
		case char >= '0' && char <= '9':
			builder.WriteRune(char)
			lastUnderscore = false
		default:
			if builder.Len() > 0 && !lastUnderscore {
				builder.WriteByte('_')
				lastUnderscore = true
			}
		}
	}
	segment := strings.Trim(builder.String(), "_")
	if segment == "" || strings.Contains(segment, "__") || !regexpSegment(segment) {
		return "", NewInvalidParamsError("invalid tool id segment", map[string]any{"segment": raw})
	}
	return segment, nil
}

func regexpSegment(segment string) bool {
	return segmentedIDPattern.MatchString(segment)
}

func isToolProviderMethod(method string) bool {
	return method == ExtensionServiceMethodProvideTools || method == ExtensionServiceMethodToolsCall
}

func ensureRPCErrorIfTyped(err error) *RPCError {
	var rpcErr *RPCError
	if errors.As(err, &rpcErr) {
		return rpcErr
	}
	return nil
}

func toolExecutionError(err error, call ExtensionToolCallRequest, sensitiveFields []string) *RPCError {
	message := redactSensitiveText(err.Error(), call.Input, sensitiveFields)
	data := map[string]any{
		"tool_id": string(call.ToolID),
		"handler": call.Handler,
		"error":   message,
	}
	if len(sensitiveFields) > 0 {
		data["input"] = redactSensitiveInput(call.Input, sensitiveFields)
		data["sensitive_input_fields"] = sensitiveFields
	}
	return NewToolExecutionError(data)
}

func redactSensitiveText(text string, input json.RawMessage, sensitiveFields []string) string {
	redacted := text
	for _, field := range sensitiveFields {
		value := readJSONPath(input, field)
		if value != "" {
			redacted = strings.ReplaceAll(redacted, value, "[REDACTED]")
		}
	}
	return redacted
}

func redactSensitiveInput(input json.RawMessage, sensitiveFields []string) any {
	if len(input) == 0 {
		return map[string]any{}
	}
	var value any
	if err := json.Unmarshal(input, &value); err != nil {
		return map[string]any{"redacted": true}
	}
	for _, field := range sensitiveFields {
		redactPath(value, strings.Split(field, "."))
	}
	return value
}

func readJSONPath(input json.RawMessage, field string) string {
	var value any
	if err := json.Unmarshal(input, &value); err != nil {
		return ""
	}
	for part := range strings.SplitSeq(field, ".") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		object, ok := value.(map[string]any)
		if !ok {
			return ""
		}
		value = object[part]
	}
	text, ok := value.(string)
	if !ok {
		return ""
	}
	return text
}

func redactPath(value any, path []string) {
	if len(path) == 0 {
		return
	}
	object, ok := value.(map[string]any)
	if !ok {
		return
	}
	head := strings.TrimSpace(path[0])
	if head == "" {
		redactPath(value, path[1:])
		return
	}
	if len(path) == 1 {
		if _, ok := object[head]; ok {
			object[head] = "[REDACTED]"
		}
		return
	}
	redactPath(object[head], path[1:])
}
