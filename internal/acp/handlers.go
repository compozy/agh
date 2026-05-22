package acp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	acpsdk "github.com/coder/acp-go-sdk"
	"github.com/pedronauck/agh/internal/store"
	toolspkg "github.com/pedronauck/agh/internal/tools"
)

const (
	sessionUpdateConfigOption = "config_option_update"
	steerDispatchTimeout      = 10 * time.Second
)

type wireSessionNotification struct {
	SessionID acpsdk.SessionId `json:"sessionId"`
	Update    json.RawMessage  `json:"update"`
}

type wireSessionUpdateEnvelope struct {
	SessionUpdate string `json:"sessionUpdate"`
}

type wirePromptResponse struct {
	StopReason acpsdk.StopReason `json:"stopReason"`
	Usage      *wireUsage        `json:"usage,omitempty"`
}

// wireNewSessionRequest keeps the workspace extension on the top-level
// session/new payload. The workspace techspec requires the JSON-RPC field name
// `additional_dirs`, even though the upstream ACP SDK does not model it yet.
type wireNewSessionRequest struct {
	Meta           any                `json:"_meta,omitempty"`
	Cwd            string             `json:"cwd"`
	McpServers     []acpsdk.McpServer `json:"mcpServers"`
	AdditionalDirs []string           `json:"additional_dirs,omitempty"`
}

// wireLoadSessionRequest mirrors session/load with the same top-level
// `additional_dirs` field name required by the workspace techspec.
type wireLoadSessionRequest struct {
	Meta           any                `json:"_meta,omitempty"`
	Cwd            string             `json:"cwd"`
	McpServers     []acpsdk.McpServer `json:"mcpServers"`
	AdditionalDirs []string           `json:"additional_dirs,omitempty"`
	SessionID      acpsdk.SessionId   `json:"sessionId"`
}

type wireUsage struct {
	InputTokens      *int64 `json:"inputTokens,omitempty"`
	OutputTokens     *int64 `json:"outputTokens,omitempty"`
	TotalTokens      *int64 `json:"totalTokens,omitempty"`
	ThoughtTokens    *int64 `json:"thoughtTokens,omitempty"`
	CacheReadTokens  *int64 `json:"cacheReadTokens,omitempty"`
	CacheWriteTokens *int64 `json:"cacheWriteTokens,omitempty"`
}

type wireUsageUpdate struct {
	SessionUpdate string    `json:"sessionUpdate"`
	Used          *int64    `json:"used,omitempty"`
	Size          *int64    `json:"size,omitempty"`
	Cost          *wireCost `json:"cost,omitempty"`
}

type wireCost struct {
	Amount   *float64 `json:"amount,omitempty"`
	Currency *string  `json:"currency,omitempty"`
}

func (p *AgentProcess) handleInbound(
	ctx context.Context,
	method string,
	params json.RawMessage,
) (any, *acpsdk.RequestError) {
	if method == acpsdk.ClientMethodSessionUpdate {
		if err := p.handleSessionUpdateWithContext(ctx, params); err != nil {
			return nil, requestError(err)
		}
		return nil, nil
	}

	switch method {
	case acpsdk.ClientMethodFsReadTextFile:
		return handleInboundRequest(ctx, params, p.handleReadTextFile)
	case acpsdk.ClientMethodFsWriteTextFile:
		return handleInboundRequest(ctx, params, p.handleWriteTextFile)
	case acpsdk.ClientMethodSessionRequestPermission:
		return handleInboundRequest(ctx, params, p.handleRequestPermission)
	case acpsdk.ClientMethodTerminalCreate:
		return handleInboundRequest(ctx, params, p.handleCreateTerminal)
	case acpsdk.ClientMethodTerminalKill:
		return handleInboundRequestNoContext(params, p.handleKillTerminal)
	case acpsdk.ClientMethodTerminalOutput:
		return handleInboundRequestNoContext(params, p.handleTerminalOutput)
	case acpsdk.ClientMethodTerminalWaitForExit:
		return handleInboundRequest(ctx, params, p.handleWaitForTerminalExit)
	case acpsdk.ClientMethodTerminalRelease:
		return handleInboundRequestNoContext(params, p.handleReleaseTerminal)
	default:
		return nil, acpsdk.NewMethodNotFound(method)
	}
}

func handleInboundRequest[Req any, Resp any](
	ctx context.Context,
	params json.RawMessage,
	fn func(context.Context, Req) (Resp, error),
) (any, *acpsdk.RequestError) {
	var request Req
	if err := json.Unmarshal(params, &request); err != nil {
		return nil, acpsdk.NewInvalidParams(map[string]any{EventTypeError: err.Error()})
	}

	response, err := fn(ctx, request)
	if err != nil {
		return nil, requestError(err)
	}
	return response, nil
}

func handleInboundRequestNoContext[Req any, Resp any](
	params json.RawMessage,
	fn func(Req) (Resp, error),
) (any, *acpsdk.RequestError) {
	var request Req
	if err := json.Unmarshal(params, &request); err != nil {
		return nil, acpsdk.NewInvalidParams(map[string]any{EventTypeError: err.Error()})
	}

	response, err := fn(request)
	if err != nil {
		return nil, requestError(err)
	}
	return response, nil
}

type readTextFileToolInput struct {
	Path  string `json:"path"`
	Line  *int   `json:"line,omitempty"`
	Limit *int   `json:"limit,omitempty"`
}

type writeTextFileToolInput struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

type createTerminalToolInput struct {
	Command         string               `json:"command"`
	Args            []string             `json:"args,omitempty"`
	Cwd             *string              `json:"cwd,omitempty"`
	Env             []acpsdk.EnvVariable `json:"env,omitempty"`
	OutputByteLimit *int                 `json:"outputByteLimit,omitempty"`
}

func (p *AgentProcess) handleReadTextFile(
	ctx context.Context,
	request acpsdk.ReadTextFileRequest,
) (acpsdk.ReadTextFileResponse, error) {
	request, err := p.interceptReadTextFileRequest(ctx, request)
	if err != nil {
		return acpsdk.ReadTextFileResponse{}, err
	}

	content, err := p.toolHostOrDefault().ReadTextFile(ctx, request.Path)
	if err != nil {
		return acpsdk.ReadTextFileResponse{}, err
	}
	return acpsdk.ReadTextFileResponse{Content: sliceLines(content, request.Line, request.Limit)}, nil
}

func (p *AgentProcess) handleWriteTextFile(
	ctx context.Context,
	request acpsdk.WriteTextFileRequest,
) (acpsdk.WriteTextFileResponse, error) {
	request, err := p.interceptWriteTextFileRequest(ctx, request)
	if err != nil {
		return acpsdk.WriteTextFileResponse{}, err
	}
	if p.isNetworkTurn() {
		return acpsdk.WriteTextFileResponse{}, ErrToolBlockedForNetworkTurn
	}
	if err := p.toolHostOrDefault().WriteTextFile(ctx, request.Path, request.Content); err != nil {
		return acpsdk.WriteTextFileResponse{}, err
	}
	return acpsdk.WriteTextFileResponse{}, nil
}

func (p *AgentProcess) handleRequestPermission(
	ctx context.Context,
	request acpsdk.RequestPermissionRequest,
) (acpsdk.RequestPermissionResponse, error) {
	turnID := p.activeTurnID()
	resource := ""
	if request.ToolCall.Title != nil {
		resource = *request.ToolCall.Title
	}
	if len(request.ToolCall.Locations) > 0 {
		resource = request.ToolCall.Locations[0].Path
	}
	title := ""
	if request.ToolCall.Title != nil {
		title = *request.ToolCall.Title
	}
	sessionID := string(request.SessionId)
	toolCallID := strings.TrimSpace(string(request.ToolCall.ToolCallId))
	requestID := p.nextPermissionRequestID(turnID, request)
	if handled, err := p.interceptProviderNativePermissionRequest(ctx, request); handled {
		switch {
		case err == nil:
		case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
			return acpsdk.RequestPermissionResponse{
				Outcome: acpsdk.NewRequestPermissionOutcomeCancelled(),
			}, nil
		case errors.Is(err, ErrPermissionDenied):
			outcome, appliedDecision := selectPermissionOutcome(request.Options, decisionRejectOnce)
			raw := buildPermissionEventRaw(requestID, appliedDecision, request)
			p.emitPermissionEvent(sessionID, turnID, requestID, title, toolCallID, resource, appliedDecision, raw)
			return acpsdk.RequestPermissionResponse{Outcome: outcome}, nil
		default:
			return acpsdk.RequestPermissionResponse{}, err
		}
	}

	decision, interactive := p.toolHostOrDefault().PermissionDecision(request)

	if !interactive {
		outcome, appliedDecision := selectPermissionOutcome(request.Options, decision)
		raw := buildPermissionEventRaw(requestID, appliedDecision, request)
		p.emitPermissionEvent(sessionID, turnID, requestID, title, toolCallID, resource, appliedDecision, raw)
		return acpsdk.RequestPermissionResponse{Outcome: outcome}, nil
	}

	requestID, pending := p.registerPendingPermission(turnID, request)
	defer p.clearPendingPermission(requestID)
	raw := buildPermissionEventRaw(requestID, decisionPending, request)
	p.emitPermissionEvent(sessionID, turnID, requestID, title, toolCallID, resource, "", raw)

	timer := time.NewTimer(p.permissionTimeoutOrDefault())
	defer timer.Stop()

	select {
	case resolvedDecision := <-pending.response:
		outcome, appliedDecision := selectPermissionOutcome(request.Options, resolvedDecision)
		raw = buildPermissionEventRaw(requestID, appliedDecision, request)
		p.emitPermissionEvent(sessionID, turnID, requestID, title, toolCallID, resource, appliedDecision, raw)
		return acpsdk.RequestPermissionResponse{Outcome: outcome}, nil
	case <-timer.C:
		outcome, appliedDecision := selectPermissionOutcome(request.Options, decisionRejectOnce)
		raw = buildPermissionEventRaw(requestID, appliedDecision, request)
		p.emitPermissionEvent(sessionID, turnID, requestID, title, toolCallID, resource, appliedDecision, raw)
		return acpsdk.RequestPermissionResponse{Outcome: outcome}, nil
	case <-ctx.Done():
		return acpsdk.RequestPermissionResponse{
			Outcome: acpsdk.NewRequestPermissionOutcomeCancelled(),
		}, nil
	}
}

func (p *AgentProcess) handleSessionUpdate(params json.RawMessage) error {
	return p.handleSessionUpdateWithContext(context.Background(), params)
}

func (p *AgentProcess) handleSessionUpdateWithContext(ctx context.Context, params json.RawMessage) error {
	var raw wireSessionNotification
	if err := json.Unmarshal(params, &raw); err != nil {
		return fmt.Errorf("acp: decode session/update notification: %w", err)
	}
	var envelope wireSessionUpdateEnvelope
	if err := json.Unmarshal(raw.Update, &envelope); err != nil {
		return fmt.Errorf("acp: decode session/update envelope: %w", err)
	}

	if envelope.SessionUpdate == "usage_update" {
		var update wireUsageUpdate
		if err := json.Unmarshal(raw.Update, &update); err != nil {
			return fmt.Errorf("acp: decode usage_update: %w", err)
		}
		usage := tokenUsageFromUsageUpdate(p.activeTurnID(), update)
		if !usage.IsZero() {
			merged := p.mergePromptUsage(usage)
			p.emitPromptEvent(AgentEvent{
				Type:      EventTypeUsage,
				SessionID: string(raw.SessionID),
				TurnID:    merged.TurnID,
				Timestamp: usage.Timestamp,
				Usage:     &merged,
				Raw:       CloneRawMessage(raw.Update),
			})
		}
		return nil
	}

	var notification acpsdk.SessionNotification
	if err := json.Unmarshal(params, &notification); err != nil {
		return fmt.Errorf("acp: decode session notification: %w", err)
	}
	if notification.Update.ConfigOptionUpdate != nil {
		p.setConfigOptions(sessionConfigOptionsFromSDK(notification.Update.ConfigOptionUpdate.ConfigOptions))
	}

	event := translateSessionUpdate(notification, raw.Update, p.activeTurnID())
	event = p.markToolEventPrechecked(event)
	p.emitPromptEvent(event)
	p.injectSteerAfterToolResult(ctx, event)
	return nil
}

func (p *AgentProcess) injectSteerAfterToolResult(ctx context.Context, boundary AgentEvent) {
	if p == nil || p.steerSource == nil || p.conn == nil || boundary.Type != EventTypeToolResult {
		return
	}
	sessionID := strings.TrimSpace(p.SessionID)
	if sessionID == "" {
		return
	}
	consumeCtx, cancel := steerConsumeContext(ctx, p)
	defer cancel()

	input, ok, err := p.steerSource.ConsumeSteer(consumeCtx, sessionID)
	if err != nil {
		p.emitPromptEvent(AgentEvent{
			Type:      EventTypeError,
			SessionID: sessionID,
			TurnID:    firstNonEmptyString(p.activeTurnID(), boundary.TurnID),
			Timestamp: timeNowUTC(),
			Error:     fmt.Sprintf("consume staged steer input: %v", err),
		})
		return
	}
	if !ok {
		return
	}

	turnID := firstNonEmptyString(p.activeTurnID(), boundary.TurnID)
	p.emitPromptEvent(AgentEvent{
		Type:      EventTypeUserMessage,
		SessionID: sessionID,
		TurnID:    turnID,
		RequestID: strings.TrimSpace(input.QueueEntryID),
		Timestamp: timeNowUTC(),
		Text:      strings.TrimSpace(input.Text),
		Action:    PromptActionSteered,
		Resource:  "session_input_queue",
		Decision:  strconv.FormatInt(input.QueueGeneration, 10),
	})

	go p.dispatchSteerPrompt(steerDispatchContext(p), input, turnID)
}

func steerConsumeContext(ctx context.Context, process *AgentProcess) (context.Context, context.CancelFunc) {
	if ctx != nil {
		detached, cancel := withoutCancelPreservingDeadline(ctx)
		if _, ok := detached.Deadline(); ok {
			return detached, cancel
		}
		timeoutCtx, timeoutCancel := context.WithTimeout(detached, steerDispatchTimeout)
		return timeoutCtx, func() {
			timeoutCancel()
			cancel()
		}
	}
	return context.WithTimeout(steerDispatchContext(process), steerDispatchTimeout)
}

func steerDispatchContext(process *AgentProcess) context.Context {
	if process != nil && process.processCtx != nil {
		return process.processCtx
	}
	return context.Background()
}

func (p *AgentProcess) dispatchSteerPrompt(ctx context.Context, input SteerInput, turnID string) {
	if p == nil || p.conn == nil {
		return
	}
	dispatchCtx, cancel := steerTimeoutContext(ctx)
	defer cancel()

	request := acpsdk.PromptRequest{
		SessionId: acpsdk.SessionId(p.SessionID),
		Prompt:    []acpsdk.ContentBlock{acpsdk.TextBlock(strings.TrimSpace(input.Text))},
	}
	if meta, err := steerPromptMeta(input); err == nil {
		request.Meta = meta
	} else {
		p.emitPromptEvent(AgentEvent{
			Type:      EventTypeError,
			SessionID: p.SessionID,
			TurnID:    turnID,
			Timestamp: timeNowUTC(),
			Error:     fmt.Sprintf("build staged steer metadata: %v", err),
		})
		return
	}
	if _, err := acpsdk.SendRequest[wirePromptResponse](
		p.conn,
		dispatchCtx,
		acpsdk.AgentMethodSessionPrompt,
		request,
	); err != nil {
		failure, _ := FailureFromError(err, store.FailurePrompt)
		p.emitPromptEvent(AgentEvent{
			Type:      EventTypeError,
			SessionID: p.SessionID,
			TurnID:    turnID,
			Timestamp: timeNowUTC(),
			Error:     fmt.Sprintf("dispatch staged steer input: %v", err),
			Failure:   failure,
			Raw:       requestErrorRaw(err),
		})
	}
}

func steerTimeoutContext(ctx context.Context) (context.Context, context.CancelFunc) {
	if ctx == nil {
		return context.WithTimeout(context.Background(), steerDispatchTimeout)
	}
	if _, ok := ctx.Deadline(); ok {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, steerDispatchTimeout)
}

func steerPromptMeta(input SteerInput) (map[string]any, error) {
	meta, err := PromptMeta{TurnSource: PromptTurnSourceUser}.ToMap()
	if err != nil {
		return nil, err
	}
	if meta == nil {
		meta = make(map[string]any)
	}
	meta["busy_input"] = map[string]any{
		"mode":             "steer",
		"queue_entry_id":   strings.TrimSpace(input.QueueEntryID),
		"queue_generation": input.QueueGeneration,
	}
	return meta, nil
}

func (p *AgentProcess) interceptReadTextFileRequest(
	ctx context.Context,
	request acpsdk.ReadTextFileRequest,
) (acpsdk.ReadTextFileRequest, error) {
	if p == nil || p.toolGateway == nil {
		request.Path = strings.TrimSpace(request.Path)
		return request, nil
	}

	input, err := json.Marshal(readTextFileToolInput{
		Path:  request.Path,
		Line:  request.Line,
		Limit: request.Limit,
	})
	if err != nil {
		return acpsdk.ReadTextFileRequest{}, fmt.Errorf("acp: marshal read_text_file input: %w", err)
	}

	patched, err := p.interceptProviderNativeTool(ctx, ToolExecutionRequest{
		ToolID:   providerNativeToolIDRead,
		ReadOnly: true,
		Input:    input,
	})
	if err != nil {
		return acpsdk.ReadTextFileRequest{}, err
	}

	var next readTextFileToolInput
	if err := json.Unmarshal(patched.Input, &next); err != nil {
		return acpsdk.ReadTextFileRequest{}, fmt.Errorf(
			"%w: invalid %s tool patch: %w",
			ErrPermissionDenied,
			providerNativeToolIDRead,
			err,
		)
	}

	request.Path = strings.TrimSpace(next.Path)
	request.Line = next.Line
	request.Limit = next.Limit
	return request, nil
}

func (p *AgentProcess) interceptWriteTextFileRequest(
	ctx context.Context,
	request acpsdk.WriteTextFileRequest,
) (acpsdk.WriteTextFileRequest, error) {
	if p == nil || p.toolGateway == nil {
		request.Path = strings.TrimSpace(request.Path)
		return request, nil
	}

	input, err := json.Marshal(writeTextFileToolInput{
		Path:    request.Path,
		Content: request.Content,
	})
	if err != nil {
		return acpsdk.WriteTextFileRequest{}, fmt.Errorf("acp: marshal write_text_file input: %w", err)
	}

	patched, err := p.interceptProviderNativeTool(ctx, ToolExecutionRequest{
		ToolID: providerNativeToolIDWrite,
		Input:  input,
	})
	if err != nil {
		return acpsdk.WriteTextFileRequest{}, err
	}

	var next writeTextFileToolInput
	if err := json.Unmarshal(patched.Input, &next); err != nil {
		return acpsdk.WriteTextFileRequest{}, fmt.Errorf(
			"%w: invalid %s tool patch: %w",
			ErrPermissionDenied,
			providerNativeToolIDWrite,
			err,
		)
	}

	request.Path = strings.TrimSpace(next.Path)
	request.Content = next.Content
	return request, nil
}

func (p *AgentProcess) interceptCreateTerminalRequest(
	ctx context.Context,
	request acpsdk.CreateTerminalRequest,
) (acpsdk.CreateTerminalRequest, error) {
	if p == nil || p.toolGateway == nil {
		request.Command = strings.TrimSpace(request.Command)
		request.Args = cloneNonEmptyStringSlice(request.Args)
		request.Cwd = cloneStringPtr(request.Cwd)
		request.Env = cloneNonEmptyEnvSlice(request.Env)
		request.OutputByteLimit = cloneIntPtr(request.OutputByteLimit)
		return request, nil
	}

	input, err := json.Marshal(createTerminalToolInput{
		Command:         request.Command,
		Args:            request.Args,
		Cwd:             request.Cwd,
		Env:             request.Env,
		OutputByteLimit: request.OutputByteLimit,
	})
	if err != nil {
		return acpsdk.CreateTerminalRequest{}, fmt.Errorf("acp: marshal terminal/create input: %w", err)
	}

	patched, err := p.interceptProviderNativeTool(ctx, ToolExecutionRequest{
		ToolID: providerNativeToolIDBash,
		Input:  input,
	})
	if err != nil {
		return acpsdk.CreateTerminalRequest{}, err
	}

	var next createTerminalToolInput
	if err := json.Unmarshal(patched.Input, &next); err != nil {
		return acpsdk.CreateTerminalRequest{}, fmt.Errorf(
			"%w: invalid %s tool patch: %w",
			ErrPermissionDenied,
			providerNativeToolIDBash,
			err,
		)
	}

	request.Command = strings.TrimSpace(next.Command)
	request.Args = append([]string(nil), next.Args...)
	request.Cwd = next.Cwd
	request.Env = append([]acpsdk.EnvVariable(nil), next.Env...)
	request.OutputByteLimit = next.OutputByteLimit
	return request, nil
}

func (p *AgentProcess) emitPermissionEvent(
	sessionID string,
	turnID string,
	requestID string,
	title string,
	toolCallID string,
	resource string,
	decision permissionDecision,
	raw json.RawMessage,
) {
	p.emitPromptEvent(AgentEvent{
		Type:       EventTypePermission,
		SessionID:  sessionID,
		TurnID:     turnID,
		RequestID:  requestID,
		Timestamp:  timeNowUTC(),
		Title:      title,
		ToolCallID: toolCallID,
		Action:     string(permissionRequestToolGrant),
		Resource:   resource,
		Decision:   string(decision),
		Raw:        CloneRawMessage(raw),
	})
}

func translateSessionUpdate(
	notification acpsdk.SessionNotification,
	rawUpdate json.RawMessage,
	turnID string,
) AgentEvent {
	event := AgentEvent{
		SessionID: string(notification.SessionId),
		TurnID:    turnID,
		Timestamp: timeNowUTC(),
		Raw:       CloneRawMessage(rawUpdate),
	}

	switch {
	case notification.Update.UserMessageChunk != nil:
		event.Type = EventTypeUserMessage
		event.Text = extractContentText(notification.Update.UserMessageChunk.Content)
	case notification.Update.AgentMessageChunk != nil:
		event.Type = EventTypeAgentMessage
		event.Text = extractContentText(notification.Update.AgentMessageChunk.Content)
	case notification.Update.AgentThoughtChunk != nil:
		event.Type = EventTypeThought
		event.Text = extractContentText(notification.Update.AgentThoughtChunk.Content)
	case notification.Update.ToolCall != nil:
		toolCall := notification.Update.ToolCall
		event.Type = EventTypeToolCall
		event.Title = toolCall.Title
		event.ToolCallID = string(toolCall.ToolCallId)
	case notification.Update.ToolCallUpdate != nil:
		toolUpdate := notification.Update.ToolCallUpdate
		event.ToolCallID = string(toolUpdate.ToolCallId)
		if toolUpdate.Title != nil {
			event.Title = *toolUpdate.Title
		}
		if toolUpdate.Status != nil &&
			(*toolUpdate.Status == acpsdk.ToolCallStatusCompleted || *toolUpdate.Status == acpsdk.ToolCallStatusFailed) {
			event.Type = EventTypeToolResult
		} else {
			event.Type = EventTypeToolCall
		}
	case notification.Update.Plan != nil:
		event.Type = EventTypePlan
	case notification.Update.AvailableCommandsUpdate != nil:
		event.Type = EventTypeSystem
		event.Title = "available_commands_update"
	case notification.Update.CurrentModeUpdate != nil:
		event.Type = EventTypeSystem
		event.Title = "current_mode_update"
	case notification.Update.ConfigOptionUpdate != nil:
		event.Type = EventTypeSystem
		event.Title = sessionUpdateConfigOption
	default:
		event.Type = EventTypeSystem
	}

	return event
}

func tokenUsageFromPromptResponse(turnID string, usage *wireUsage) TokenUsage {
	if usage == nil {
		return TokenUsage{TurnID: turnID}
	}
	return TokenUsage{
		TurnID:           turnID,
		InputTokens:      usage.InputTokens,
		OutputTokens:     usage.OutputTokens,
		TotalTokens:      usage.TotalTokens,
		ThoughtTokens:    usage.ThoughtTokens,
		CacheReadTokens:  usage.CacheReadTokens,
		CacheWriteTokens: usage.CacheWriteTokens,
		Timestamp:        timeNowUTC(),
	}
}

func tokenUsageFromUsageUpdate(turnID string, update wireUsageUpdate) TokenUsage {
	var amount *float64
	var currency *string
	if update.Cost != nil {
		amount = update.Cost.Amount
		currency = update.Cost.Currency
	}
	return TokenUsage{
		TurnID:       turnID,
		ContextUsed:  update.Used,
		ContextSize:  update.Size,
		CostAmount:   amount,
		CostCurrency: currency,
		Timestamp:    timeNowUTC(),
	}
}

func requestError(err error) *acpsdk.RequestError {
	if err == nil {
		return nil
	}
	if requestErr, ok := errors.AsType[*acpsdk.RequestError](err); ok {
		return requestErr
	}
	if data, ok := toolErrorRequestData(err); ok {
		return acpsdk.NewInternalError(data)
	}
	if errors.Is(err, ErrPermissionDenied) || errors.Is(err, ErrInvalidPath) ||
		errors.Is(err, ErrPathOutsideWorkspace) ||
		errors.Is(err, ErrToolBlockedForNetworkTurn) {
		return acpsdk.NewInvalidParams(map[string]any{EventTypeError: err.Error()})
	}
	return acpsdk.NewInternalError(map[string]any{EventTypeError: err.Error()})
}

func toolErrorRequestData(err error) (map[string]any, bool) {
	var toolErr *toolspkg.ToolError
	if errors.As(err, &toolErr) && toolErr != nil {
		data := map[string]any{
			EventTypeError: err.Error(),
			"tool_code":    string(toolErr.Code),
		}
		if string(toolErr.ToolID) != "" {
			data["tool_id"] = string(toolErr.ToolID)
		}
		if len(toolErr.ReasonCodes) > 0 {
			data["reason_codes"] = reasonCodesAsStrings(toolErr.ReasonCodes)
		}
		return data, true
	}
	if validation, ok := errors.AsType[*toolspkg.ValidationError](err); ok {
		data := map[string]any{
			EventTypeError: err.Error(),
			"reason":       string(validation.Reason),
		}
		if strings.TrimSpace(validation.Field) != "" {
			data["field"] = strings.TrimSpace(validation.Field)
		}
		return data, true
	}
	return nil, false
}

func reasonCodesAsStrings(reasons []toolspkg.ReasonCode) []string {
	values := make([]string, 0, len(reasons))
	for _, reason := range reasons {
		if trimmed := strings.TrimSpace(string(reason)); trimmed != "" {
			values = append(values, trimmed)
		}
	}
	return values
}

func sliceLines(content string, line, limit *int) string {
	if line == nil && limit == nil {
		return content
	}

	lines := strings.Split(content, "\n")
	start := 0
	if line != nil && *line > 1 {
		start = min(*line-1, len(lines))
	}
	end := len(lines)
	if limit != nil && *limit >= 0 && start+*limit < end {
		end = start + *limit
	}
	return strings.Join(lines[start:end], "\n")
}

func extractContentText(block acpsdk.ContentBlock) string {
	switch {
	case block.Text != nil:
		return block.Text.Text
	case block.ResourceLink != nil:
		return block.ResourceLink.Uri
	default:
		return ""
	}
}

func fallbackPermissionEventRaw(requestID string, decision permissionDecision) json.RawMessage {
	var builder strings.Builder
	builder.WriteString(`{"request_id":`)
	builder.WriteString(strconv.Quote(requestID))
	if decision != "" && decision != decisionPending {
		builder.WriteString(`,"decision":`)
		builder.WriteString(strconv.Quote(string(decision)))
	}
	builder.WriteByte('}')
	return json.RawMessage(builder.String())
}

func mustMarshalJSON(value any) json.RawMessage {
	if value == nil {
		return nil
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return nil
	}
	return encoded
}

func timeNowUTC() time.Time {
	return time.Now().UTC()
}

func (p *AgentProcess) activeTurnID() string {
	p.promptMu.RLock()
	defer p.promptMu.RUnlock()
	active := p.activePrompt
	if active == nil {
		return ""
	}
	return active.turnID
}
