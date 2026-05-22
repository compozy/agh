package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"

	aghconfig "github.com/compozy/agh/internal/config"
	"github.com/compozy/agh/internal/diagnostics"
	mcpauth "github.com/compozy/agh/internal/mcp/auth"
	toolspkg "github.com/compozy/agh/internal/tools"
	"github.com/compozy/agh/internal/vault"
	mcpclient "github.com/mark3labs/mcp-go/client"
	mcptransport "github.com/mark3labs/mcp-go/client/transport"
	mcpsdk "github.com/mark3labs/mcp-go/mcp"
)

const (
	executorBearerValue = "Bearer"
	executorMCPKey      = "mcp"
)

const (
	defaultCallTimeout      = 30 * time.Second
	toolResultIsErrorKey    = "is_error"
	authStatusRefreshFailed = "refresh_failed"
)

// ServerResolver returns the current daemon-visible MCP server configuration.
type ServerResolver interface {
	ResolveMCPServers(ctx context.Context) ([]aghconfig.MCPServer, error)
}

// ServerResolverFunc adapts a function into a server resolver.
type ServerResolverFunc func(context.Context) ([]aghconfig.MCPServer, error)

// ResolveMCPServers returns the configured MCP servers.
func (f ServerResolverFunc) ResolveMCPServers(ctx context.Context) ([]aghconfig.MCPServer, error) {
	if f == nil {
		return nil, nil
	}
	return f(ctx)
}

type authService interface {
	Status(ctx context.Context, cfg mcpauth.ServerConfig) (mcpauth.Status, error)
	Refresh(ctx context.Context, cfg mcpauth.ServerConfig) (mcpauth.Status, error)
}

type secretRefResolver interface {
	ResolveRef(ctx context.Context, ref string) (string, error)
}

// CallExecutor lists and calls configured MCP servers through mcp-go clients.
type CallExecutor struct {
	servers        ServerResolver
	tokenStore     mcpauth.TokenStore
	auth           authService
	lookupSecret   func(string) string
	secretResolver secretRefResolver
	httpClient     *http.Client
	timeout        time.Duration
}

var _ toolspkg.MCPCallExecutor = (*CallExecutor)(nil)
var _ toolspkg.MCPAuthStatusProvider = (*CallExecutor)(nil)

// CallExecutorOption configures the daemon-owned MCP executor.
type CallExecutorOption func(*CallExecutor)

// WithTokenStore allows remote MCP authorization headers to be injected inside internal/mcp.
func WithTokenStore(store mcpauth.TokenStore) CallExecutorOption {
	return func(executor *CallExecutor) {
		executor.tokenStore = store
	}
}

// WithSecretLookup resolves auth client secret environment-variable names.
func WithSecretLookup(lookup func(string) string) CallExecutorOption {
	return func(executor *CallExecutor) {
		executor.lookupSecret = lookup
	}
}

// WithSecretResolver resolves env: and vault: refs for MCP auth and stdio secret_env launch bindings.
func WithSecretResolver(resolver secretRefResolver) CallExecutorOption {
	return func(executor *CallExecutor) {
		executor.secretResolver = resolver
	}
}

// WithHTTPClient configures remote MCP and auth HTTP calls with an explicit client.
func WithHTTPClient(client *http.Client) CallExecutorOption {
	return func(executor *CallExecutor) {
		executor.httpClient = client
	}
}

// WithTimeout configures the default bounded call timeout when a caller has no deadline.
func WithTimeout(timeout time.Duration) CallExecutorOption {
	return func(executor *CallExecutor) {
		if timeout > 0 {
			executor.timeout = timeout
		}
	}
}

func withAuthService(service authService) CallExecutorOption {
	return func(executor *CallExecutor) {
		executor.auth = service
	}
}

// NewMCPCallExecutor constructs the daemon-owned MCP executor.
func NewMCPCallExecutor(
	servers ServerResolver,
	opts ...CallExecutorOption,
) (*CallExecutor, error) {
	if servers == nil {
		return nil, toolspkg.NewValidationError(
			"servers",
			toolspkg.ReasonDependencyMissing,
			"mcp server resolver is required",
		)
	}
	executor := &CallExecutor{
		servers: servers,
		httpClient: &http.Client{
			Timeout: defaultCallTimeout,
		},
		timeout: defaultCallTimeout,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(executor)
		}
	}
	if executor.lookupSecret == nil {
		executor.lookupSecret = func(string) string { return "" }
	}
	if executor.timeout <= 0 {
		executor.timeout = defaultCallTimeout
	}
	if executor.httpClient == nil {
		executor.httpClient = &http.Client{Timeout: executor.timeout}
	}
	if executor.httpClient.Timeout <= 0 {
		cloned := *executor.httpClient
		cloned.Timeout = executor.timeout
		executor.httpClient = &cloned
	}
	if executor.auth == nil && executor.tokenStore != nil {
		service, err := mcpauth.NewService(executor.tokenStore, mcpauth.WithHTTPClient(executor.httpClient))
		if err != nil {
			return nil, fmt.Errorf("mcp: create auth service: %w", err)
		}
		executor.auth = service
	}
	return executor, nil
}

// ListTools discovers tools from one configured MCP server.
func (e *CallExecutor) ListTools(
	ctx context.Context,
	source toolspkg.SourceRef,
) ([]toolspkg.MCPToolDescriptor, error) {
	if e == nil {
		return nil, toolspkg.NewValidationError(
			"executor",
			toolspkg.ReasonDependencyMissing,
			"mcp executor is required",
		)
	}
	if ctx == nil {
		return nil, toolspkg.NewToolError(
			toolspkg.ErrorCodeCanceled,
			"",
			"mcp call context is required",
			toolspkg.ErrToolCanceled,
			toolspkg.ReasonCallCanceled,
		)
	}
	ctx, cancel := e.callContext(ctx)
	defer cancel()
	server, err := e.resolveServer(ctx, source)
	if err != nil {
		return nil, err
	}
	if err := e.ensureAuthorized(ctx, server); err != nil {
		return nil, err
	}
	client, err := e.openClient(ctx, server)
	if err != nil {
		return nil, err
	}
	defer closeMCPClient(client)
	if err := initializeClient(ctx, client); err != nil {
		return nil, normalizeMCPError("", err)
	}
	result, err := client.ListTools(ctx, mcpsdk.ListToolsRequest{})
	if err != nil {
		return nil, normalizeMCPError("", err)
	}
	descriptors := make([]toolspkg.MCPToolDescriptor, 0, len(result.Tools))
	for i := range result.Tools {
		descriptor, err := e.descriptorFromTool(source, server, result.Tools[i])
		if err != nil {
			return nil, fmt.Errorf("mcp: normalize tool %q: %w", result.Tools[i].Name, err)
		}
		descriptors = append(descriptors, descriptor)
	}
	return descriptors, nil
}

// CallTool invokes one configured MCP tool.
func (e *CallExecutor) CallTool(
	ctx context.Context,
	source toolspkg.SourceRef,
	req toolspkg.MCPToolCallRequest,
) (toolspkg.ToolResult, error) {
	if e == nil {
		return toolspkg.ToolResult{}, toolspkg.NewValidationError(
			"executor",
			toolspkg.ReasonDependencyMissing,
			"mcp executor is required",
		)
	}
	if ctx == nil {
		return toolspkg.ToolResult{}, toolspkg.NewToolError(
			toolspkg.ErrorCodeCanceled,
			req.ToolID,
			"mcp call context is required",
			toolspkg.ErrToolCanceled,
			toolspkg.ReasonCallCanceled,
		)
	}
	ctx, cancel := e.callContext(ctx)
	defer cancel()
	server, err := e.resolveServer(ctx, source)
	if err != nil {
		return toolspkg.ToolResult{}, err
	}
	if err := e.ensureAuthorized(ctx, server); err != nil {
		return toolspkg.ToolResult{}, err
	}
	client, err := e.openClient(ctx, server)
	if err != nil {
		return toolspkg.ToolResult{}, err
	}
	defer closeMCPClient(client)
	if err := initializeClient(ctx, client); err != nil {
		return toolspkg.ToolResult{}, normalizeMCPError(req.ToolID, err)
	}
	arguments, err := decodeArguments(req.Input)
	if err != nil {
		return toolspkg.ToolResult{}, err
	}
	result, err := client.CallTool(ctx, mcpsdk.CallToolRequest{
		Params: mcpsdk.CallToolParams{
			Name:      strings.TrimSpace(req.RawToolName),
			Arguments: arguments,
		},
	})
	if err != nil {
		return toolspkg.ToolResult{}, normalizeMCPError(req.ToolID, err)
	}
	return toolResultFromMCP(result)
}

// Status returns token-redacted auth diagnostics for registry availability.
func (e *CallExecutor) Status(
	ctx context.Context,
	source toolspkg.SourceRef,
) (toolspkg.MCPAuthStatus, error) {
	if e == nil {
		return toolspkg.MCPAuthStatus{}, toolspkg.NewValidationError(
			"executor",
			toolspkg.ReasonDependencyMissing,
			"mcp executor is required",
		)
	}
	if ctx == nil {
		return toolspkg.MCPAuthStatus{}, toolspkg.NewToolError(
			toolspkg.ErrorCodeCanceled,
			"",
			"mcp status context is required",
			toolspkg.ErrToolCanceled,
			toolspkg.ReasonCallCanceled,
		)
	}
	server, err := e.resolveServer(ctx, source)
	if err != nil {
		return toolspkg.MCPAuthStatus{}, err
	}
	return e.authStatus(ctx, server)
}

func (e *CallExecutor) callContext(ctx context.Context) (context.Context, context.CancelFunc) {
	if _, ok := ctx.Deadline(); ok {
		return context.WithCancel(ctx)
	}
	return context.WithTimeout(ctx, e.timeout)
}

func (e *CallExecutor) resolveServer(
	ctx context.Context,
	source toolspkg.SourceRef,
) (aghconfig.MCPServer, error) {
	if ctx == nil {
		return aghconfig.MCPServer{}, toolspkg.NewToolError(
			toolspkg.ErrorCodeCanceled,
			"",
			"mcp server resolution context is required",
			toolspkg.ErrToolCanceled,
			toolspkg.ReasonCallCanceled,
		)
	}
	servers, err := e.servers.ResolveMCPServers(ctx)
	if err != nil {
		return aghconfig.MCPServer{}, fmt.Errorf("mcp: resolve configured servers: %w", err)
	}
	target := strings.TrimSpace(firstNonEmpty(source.RawServerName, source.Owner))
	for _, server := range servers {
		if mcpServerMatches(server, target) || mcpServerMatches(server, source.Owner) {
			return cloneMCPServer(server), nil
		}
	}
	return aghconfig.MCPServer{}, toolspkg.NewToolError(
		toolspkg.ErrorCodeUnavailable,
		"",
		fmt.Sprintf("mcp server %q is unavailable", target),
		toolspkg.ErrToolUnavailable,
		toolspkg.ReasonMCPUnreachable,
	)
}

func (e *CallExecutor) openClient(ctx context.Context, server aghconfig.MCPServer) (*mcpclient.Client, error) {
	transport := server.EffectiveTransport()
	switch transport {
	case aghconfig.MCPServerTransportStdio:
		env, err := e.mcpServerEnv(ctx, server)
		if err != nil {
			return nil, err
		}
		client, err := mcpclient.NewStdioMCPClientWithOptions(
			strings.TrimSpace(server.Command),
			env,
			trimStrings(server.Args),
			mcptransport.WithCommandFunc(mcpStdioCommandWithExactEnv),
		)
		if err != nil {
			return nil, normalizeMCPError("", err)
		}
		if err := client.Start(ctx); err != nil {
			return nil, normalizeMCPStartError(client, err)
		}
		return client, nil
	case aghconfig.MCPServerTransportHTTP:
		options := []mcptransport.StreamableHTTPCOption{
			mcptransport.WithHTTPBasicClient(e.httpClientForCall()),
		}
		if server.Auth.Enabled() {
			options = append(options, mcptransport.WithHTTPHeaderFunc(e.authHeaderFunc(server)))
		}
		client, err := mcpclient.NewStreamableHttpClient(strings.TrimSpace(server.URL), options...)
		if err != nil {
			return nil, normalizeMCPError("", err)
		}
		if err := client.Start(ctx); err != nil {
			return nil, normalizeMCPStartError(client, err)
		}
		return client, nil
	case aghconfig.MCPServerTransportSSE:
		options := []mcptransport.ClientOption{
			mcptransport.WithHTTPClient(e.httpClientForCall()),
			mcptransport.WithEndpointTimeout(e.timeout),
			mcptransport.WithResponseTimeout(e.timeout),
		}
		if server.Auth.Enabled() {
			options = append(options, mcptransport.WithHeaderFunc(e.authHeaderFunc(server)))
		}
		client, err := mcpclient.NewSSEMCPClient(strings.TrimSpace(server.URL), options...)
		if err != nil {
			return nil, normalizeMCPError("", err)
		}
		if err := client.Start(ctx); err != nil {
			return nil, normalizeMCPStartError(client, err)
		}
		return client, nil
	default:
		return nil, toolspkg.NewValidationError(
			"mcp_server.transport",
			toolspkg.ReasonMCPUnreachable,
			"unsupported mcp transport",
		)
	}
}

func initializeClient(ctx context.Context, client *mcpclient.Client) error {
	_, err := client.Initialize(ctx, mcpsdk.InitializeRequest{
		Params: mcpsdk.InitializeParams{
			ProtocolVersion: mcpsdk.LATEST_PROTOCOL_VERSION,
			ClientInfo: mcpsdk.Implementation{
				Name:    "agh",
				Version: "0.0.0",
			},
			Capabilities: mcpsdk.ClientCapabilities{},
		},
	})
	return err
}

func (e *CallExecutor) ensureAuthorized(ctx context.Context, server aghconfig.MCPServer) error {
	if server.EffectiveTransport() == aghconfig.MCPServerTransportStdio || !server.Auth.Enabled() {
		return nil
	}
	status, err := e.authStatus(ctx, server)
	if err != nil {
		return err
	}
	switch status.Status {
	case string(mcpauth.StatusAuthenticated):
		return nil
	case string(mcpauth.StatusExpired):
		refreshed, refreshErr := e.refreshAuth(ctx, server)
		if refreshErr != nil {
			return toolspkg.NewToolError(
				toolspkg.ErrorCodeUnavailable,
				"",
				"mcp auth refresh failed",
				toolspkg.ErrToolUnavailable,
				toolspkg.ReasonMCPAuthRefreshFailed,
			)
		}
		if refreshed.Status == string(mcpauth.StatusAuthenticated) {
			return nil
		}
		reason, ok := toolspkg.MCPAuthStatusReason(refreshed)
		if !ok {
			reason = toolspkg.ReasonMCPAuthRefreshFailed
		}
		return unavailableAuthError(reason)
	default:
		reason, ok := toolspkg.MCPAuthStatusReason(status)
		if !ok {
			reason = toolspkg.ReasonMCPAuthRequired
		}
		if reason == toolspkg.ReasonMCPAuthUnconfigured {
			reason = toolspkg.ReasonMCPAuthRequired
		}
		return unavailableAuthError(reason)
	}
}

func (e *CallExecutor) authStatus(
	ctx context.Context,
	server aghconfig.MCPServer,
) (toolspkg.MCPAuthStatus, error) {
	cfg, err := mcpauth.ServerConfigFromMCP(ctx, server, e.resolveSecretRef)
	if err != nil {
		return toolspkg.MCPAuthStatus{}, fmt.Errorf("mcp: build auth config: %w", err)
	}
	if !server.Auth.Enabled() {
		return redactedAuthStatus(mcpauth.Status{
			ServerName: server.Name,
			Status:     mcpauth.StatusUnconfigured,
			RemoteURL:  server.URL,
		}), nil
	}
	if e.auth == nil {
		return toolspkg.MCPAuthStatus{}, toolspkg.NewValidationError(
			"auth",
			toolspkg.ReasonDependencyMissing,
			"mcp auth service is required for auth-enabled servers",
		)
	}
	status, err := e.auth.Status(ctx, cfg)
	if err != nil {
		return toolspkg.MCPAuthStatus{}, fmt.Errorf("mcp: read auth status: %w", err)
	}
	return redactedAuthStatus(status), nil
}

func (e *CallExecutor) refreshAuth(
	ctx context.Context,
	server aghconfig.MCPServer,
) (toolspkg.MCPAuthStatus, error) {
	if e.auth == nil {
		return toolspkg.MCPAuthStatus{}, toolspkg.NewValidationError(
			"auth",
			toolspkg.ReasonDependencyMissing,
			"mcp auth service is required for refresh",
		)
	}
	cfg, err := mcpauth.ServerConfigFromMCP(ctx, server, e.resolveSecretRef)
	if err != nil {
		return toolspkg.MCPAuthStatus{}, fmt.Errorf("mcp: build auth config: %w", err)
	}
	status, err := e.auth.Refresh(ctx, cfg)
	if err != nil {
		redacted := redactedAuthStatus(mcpauth.Status{
			ServerName:   cfg.ServerName,
			Status:       mcpauth.StatusInvalid,
			AuthType:     cfg.Type,
			ClientID:     cfg.ClientID,
			Scopes:       cfg.Scopes,
			Diagnostic:   "refresh failed",
			TokenPresent: false,
		})
		redacted.Status = authStatusRefreshFailed
		return redacted, err
	}
	return redactedAuthStatus(status), nil
}

func (e *CallExecutor) authHeaderFunc(server aghconfig.MCPServer) mcptransport.HTTPHeaderFunc {
	return func(ctx context.Context) map[string]string {
		header := e.authorizationHeader(ctx, server)
		if header == "" {
			return nil
		}
		return map[string]string{"Authorization": header}
	}
}

func (e *CallExecutor) authorizationHeader(ctx context.Context, server aghconfig.MCPServer) string {
	if e == nil || e.tokenStore == nil {
		return ""
	}
	token, err := e.tokenStore.GetMCPAuthToken(ctx, strings.TrimSpace(server.Name))
	if err != nil || strings.TrimSpace(token.AccessToken) == "" {
		return ""
	}
	diagnostics.RegisterDynamicSecret(token.AccessToken)
	tokenType := strings.TrimSpace(token.TokenType)
	if tokenType == "" {
		tokenType = executorBearerValue
	}
	if !strings.EqualFold(tokenType, executorBearerValue) {
		return ""
	}
	return "Bearer " + strings.TrimSpace(token.AccessToken)
}

func (e *CallExecutor) descriptorFromTool(
	source toolspkg.SourceRef,
	server aghconfig.MCPServer,
	tool mcpsdk.Tool,
) (toolspkg.MCPToolDescriptor, error) {
	id, err := toolspkg.Canonicalize(server.Name, tool.Name)
	if err != nil {
		return toolspkg.MCPToolDescriptor{}, err
	}
	owner, err := mcpOwner(id)
	if err != nil {
		return toolspkg.MCPToolDescriptor{}, err
	}
	inputSchema, err := inputSchemaBytes(tool)
	if err != nil {
		return toolspkg.MCPToolDescriptor{}, err
	}
	outputSchema, err := outputSchemaBytes(tool)
	if err != nil {
		return toolspkg.MCPToolDescriptor{}, err
	}
	mcpSource := toolspkg.SourceRef{
		Kind:            toolspkg.SourceMCP,
		Owner:           owner,
		RawServerName:   strings.TrimSpace(server.Name),
		RawToolName:     strings.TrimSpace(tool.Name),
		ResourceID:      source.ResourceID,
		ResourceVersion: source.ResourceVersion,
		WorkspaceID:     source.WorkspaceID,
		Scope:           source.Scope,
	}
	readOnly := tool.Annotations.ReadOnlyHint != nil && *tool.Annotations.ReadOnlyHint
	return toolspkg.MCPToolDescriptor{
		ID:           id,
		RawName:      strings.TrimSpace(tool.Name),
		Title:        strings.TrimSpace(tool.Annotations.Title),
		Description:  strings.TrimSpace(tool.Description),
		InputSchema:  inputSchema,
		OutputSchema: outputSchema,
		Source:       mcpSource,
		ReadOnly:     readOnly,
	}, nil
}

func inputSchemaBytes(tool mcpsdk.Tool) (json.RawMessage, error) {
	if len(tool.RawInputSchema) > 0 {
		return cloneRaw(tool.RawInputSchema), nil
	}
	if strings.TrimSpace(tool.InputSchema.Type) == "" {
		return nil, toolspkg.NewValidationError(
			"input_schema",
			toolspkg.ReasonSchemaInvalid,
			"mcp input schema is missing",
		)
	}
	data, err := json.Marshal(tool.InputSchema)
	if err != nil {
		return nil, fmt.Errorf("mcp: encode input schema: %w", err)
	}
	return json.RawMessage(data), nil
}

func outputSchemaBytes(tool mcpsdk.Tool) (json.RawMessage, error) {
	if len(tool.RawOutputSchema) > 0 {
		return cloneRaw(tool.RawOutputSchema), nil
	}
	if strings.TrimSpace(tool.OutputSchema.Type) == "" {
		return nil, nil
	}
	data, err := json.Marshal(tool.OutputSchema)
	if err != nil {
		return nil, fmt.Errorf("mcp: encode output schema: %w", err)
	}
	return json.RawMessage(data), nil
}

func toolResultFromMCP(result *mcpsdk.CallToolResult) (toolspkg.ToolResult, error) {
	if result == nil {
		return toolspkg.ToolResult{}, nil
	}
	content := make([]toolspkg.ToolContent, 0, len(result.Content))
	for i := range result.Content {
		converted, err := toolContentFromMCP(result.Content[i])
		if err != nil {
			return toolspkg.ToolResult{}, err
		}
		content = append(content, converted)
	}
	var structured json.RawMessage
	if result.StructuredContent != nil {
		data, err := json.Marshal(result.StructuredContent)
		if err != nil {
			return toolspkg.ToolResult{}, fmt.Errorf("mcp: encode structured content: %w", err)
		}
		structured = data
	}
	metadata := map[string]json.RawMessage{}
	if result.IsError {
		metadata[toolResultIsErrorKey] = json.RawMessage(`true`)
	}
	if len(metadata) == 0 {
		metadata = nil
	}
	return toolspkg.ToolResult{
		Content:    content,
		Structured: structured,
		Preview:    mcpPreview(content, result.IsError),
		Metadata:   metadata,
	}, nil
}

func toolContentFromMCP(content mcpsdk.Content) (toolspkg.ToolContent, error) {
	switch typed := content.(type) {
	case mcpsdk.TextContent:
		return toolspkg.ToolContent{Type: typed.Type, Text: typed.Text}, nil
	case mcpsdk.ImageContent:
		data, err := json.Marshal(typed.Data)
		if err != nil {
			return toolspkg.ToolContent{}, fmt.Errorf("mcp: encode image content: %w", err)
		}
		return toolspkg.ToolContent{Type: typed.Type, Data: data, MIMEType: typed.MIMEType}, nil
	case mcpsdk.AudioContent:
		data, err := json.Marshal(typed.Data)
		if err != nil {
			return toolspkg.ToolContent{}, fmt.Errorf("mcp: encode audio content: %w", err)
		}
		return toolspkg.ToolContent{Type: typed.Type, Data: data, MIMEType: typed.MIMEType}, nil
	default:
		data, err := json.Marshal(content)
		if err != nil {
			return toolspkg.ToolContent{}, fmt.Errorf("mcp: encode content block: %w", err)
		}
		return toolspkg.ToolContent{Type: executorMCPKey, Data: data}, nil
	}
}

func mcpPreview(content []toolspkg.ToolContent, isError bool) string {
	prefix := ""
	if isError {
		prefix = "error: "
	}
	for _, item := range content {
		if strings.TrimSpace(item.Text) != "" {
			return prefix + strings.TrimSpace(item.Text)
		}
	}
	if len(content) == 0 {
		return prefix + "empty MCP result"
	}
	return fmt.Sprintf("%s%d MCP content blocks", prefix, len(content))
}

func decodeArguments(raw json.RawMessage) (any, error) {
	if strings.TrimSpace(string(raw)) == "" {
		return map[string]any{}, nil
	}
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil, toolspkg.NewToolError(
			toolspkg.ErrorCodeInvalidInput,
			"",
			"mcp tool input is invalid JSON",
			fmt.Errorf("%w: %w", toolspkg.ErrToolInvalidInput, err),
			toolspkg.ReasonSchemaInvalid,
		)
	}
	return value, nil
}

func normalizeMCPError(id toolspkg.ToolID, err error) error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, context.Canceled):
		return toolspkg.NewToolError(
			toolspkg.ErrorCodeCanceled,
			id,
			"mcp call canceled",
			toolspkg.ErrToolCanceled,
			toolspkg.ReasonCallCanceled,
		)
	case errors.Is(err, context.DeadlineExceeded):
		return toolspkg.NewToolError(
			toolspkg.ErrorCodeTimedOut,
			id,
			"mcp call timed out",
			toolspkg.ErrToolTimedOut,
			toolspkg.ReasonCallTimedOut,
		)
	case errors.Is(err, mcptransport.ErrAuthorizationRequired):
		return unavailableAuthError(toolspkg.ReasonMCPAuthRequired)
	default:
		return fmt.Errorf("mcp: call upstream server: %w", err)
	}
}

func unavailableAuthError(reason toolspkg.ReasonCode) error {
	return toolspkg.NewToolError(
		toolspkg.ErrorCodeUnavailable,
		"",
		"mcp authentication is unavailable",
		toolspkg.ErrToolUnavailable,
		reason,
	)
}

func redactedAuthStatus(status mcpauth.Status) toolspkg.MCPAuthStatus {
	var expiresAt *time.Time
	if status.ExpiresAt != nil {
		cloned := status.ExpiresAt.UTC()
		expiresAt = &cloned
	}
	return toolspkg.MCPAuthStatus{
		ServerName:   strings.TrimSpace(status.ServerName),
		Status:       strings.TrimSpace(string(status.Status)),
		AuthType:     strings.TrimSpace(status.AuthType),
		ClientID:     strings.TrimSpace(status.ClientID),
		Scopes:       trimStrings(status.Scopes),
		ExpiresAt:    expiresAt,
		Refreshable:  status.Refreshable,
		TokenPresent: status.TokenPresent,
		Diagnostic:   strings.TrimSpace(status.Diagnostic),
	}
}

func normalizeMCPStartError(client *mcpclient.Client, err error) error {
	if closeErr := client.Close(); closeErr != nil {
		err = fmt.Errorf("%w; close MCP client after start failure: %v", err, closeErr)
	}
	return normalizeMCPError("", err)
}

func closeMCPClient(client *mcpclient.Client) {
	if client == nil {
		return
	}
	if err := client.Close(); err != nil {
		// The operation already completed; callers have no useful recovery action for transport close failure.
		return
	}
}

func (e *CallExecutor) httpClientForCall() *http.Client {
	cloned := *e.httpClient
	if cloned.Timeout <= 0 {
		cloned.Timeout = e.timeout
	}
	return &cloned
}

func mcpServerMatches(server aghconfig.MCPServer, target string) bool {
	target = strings.TrimSpace(target)
	if target == "" {
		return false
	}
	if strings.TrimSpace(server.Name) == target {
		return true
	}
	id, err := toolspkg.Canonicalize(server.Name, "tool")
	if err != nil {
		return false
	}
	owner, err := mcpOwner(id)
	return err == nil && owner == target
}

func mcpOwner(id toolspkg.ToolID) (string, error) {
	segments, err := id.Segments()
	if err != nil {
		return "", err
	}
	if len(segments) != 3 {
		return "", toolspkg.NewValidationError(
			"tool_id",
			toolspkg.ReasonIDInvalidFormat,
			"mcp tool id must contain namespace server and tool segments",
		)
	}
	return segments[1], nil
}

func mcpServerEnv(env map[string]string) []string {
	merged := mcpStdioBaseEnv()
	for key, value := range env {
		trimmedKey := strings.TrimSpace(key)
		if trimmedKey == "" {
			continue
		}
		merged[trimmedKey] = value
	}
	keys := make([]string, 0, len(merged))
	for key := range merged {
		if strings.TrimSpace(key) != "" {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	values := make([]string, 0, len(keys))
	for _, key := range keys {
		values = append(values, strings.TrimSpace(key)+"="+merged[key])
	}
	return values
}

func mcpStdioCommandWithExactEnv(ctx context.Context, command string, env []string, args []string) (*exec.Cmd, error) {
	cmd := exec.CommandContext(ctx, command, args...)
	cmd.Env = append([]string(nil), env...)
	return cmd, nil
}

func mcpStdioBaseEnv() map[string]string {
	keys := []string{
		"PATH",
		"HOME",
		"TMPDIR",
		"TMP",
		"TEMP",
		"SystemRoot",
		"WINDIR",
		"COMSPEC",
		"PATHEXT",
		"SSL_CERT_FILE",
		"SSL_CERT_DIR",
	}
	base := make(map[string]string, len(keys))
	for _, key := range keys {
		if value, ok := os.LookupEnv(key); ok {
			base[key] = value
		}
	}
	return base
}

func (e *CallExecutor) mcpServerEnv(ctx context.Context, server aghconfig.MCPServer) ([]string, error) {
	env := cloneStringMap(server.Env)
	if len(server.SecretEnv) > 0 {
		if env == nil {
			env = make(map[string]string, len(server.SecretEnv))
		}
		for key, ref := range server.SecretEnv {
			trimmedKey := strings.TrimSpace(key)
			if trimmedKey == "" {
				continue
			}
			value, err := e.resolveSecretRef(ctx, ref)
			if err != nil {
				return nil, fmt.Errorf("mcp: resolve secret_env %s for server %q: %w", trimmedKey, server.Name, err)
			}
			diagnostics.RegisterDynamicSecret(value)
			env[trimmedKey] = value
		}
	}
	return mcpServerEnv(env), nil
}

func (e *CallExecutor) resolveSecretRef(ctx context.Context, ref string) (string, error) {
	normalized := vault.NormalizeRef(ref)
	if normalized == "" {
		return "", nil
	}
	if e != nil && e.secretResolver != nil {
		value, err := e.secretResolver.ResolveRef(ctx, normalized)
		if err != nil {
			return "", err
		}
		if strings.TrimSpace(value) != "" {
			diagnostics.RegisterDynamicSecret(value)
		}
		return value, nil
	}
	if vault.IsEnvRef(normalized) {
		envName, err := vault.EnvNameFromRef(normalized)
		if err != nil {
			return "", err
		}
		if e != nil && e.lookupSecret != nil {
			value := e.lookupSecret(envName)
			if strings.TrimSpace(value) == "" {
				return "", fmt.Errorf("%w: env:%s", vault.ErrMissingSecret, envName)
			}
			diagnostics.RegisterDynamicSecret(value)
			return value, nil
		}
	}
	return "", fmt.Errorf("%w: %s", vault.ErrUnsupportedSecretRef, normalized)
}

func cloneMCPServer(server aghconfig.MCPServer) aghconfig.MCPServer {
	server.Args = append([]string(nil), server.Args...)
	server.Env = cloneStringMap(server.Env)
	server.SecretEnv = cloneStringMap(server.SecretEnv)
	server.Auth.Scopes = append([]string(nil), server.Auth.Scopes...)
	return server
}

func cloneStringMap(src map[string]string) map[string]string {
	if len(src) == 0 {
		return nil
	}
	cloned := make(map[string]string, len(src))
	maps.Copy(cloned, src)
	return cloned
}

func cloneRaw(src json.RawMessage) json.RawMessage {
	if len(src) == 0 {
		return nil
	}
	return append(json.RawMessage(nil), src...)
}

func trimStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	out := make([]string, 0, len(values))
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
