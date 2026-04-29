package tools

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"time"
)

type dispatchTarget struct {
	descriptor Descriptor
	handle     Handle
	view       ToolView
}

func (r *RuntimeRegistry) dispatch(ctx context.Context, scope Scope, req CallRequest) (ToolResult, error) {
	started := time.Now().UTC()
	if err := contextErr(ctx, req.ToolID); err != nil {
		return ToolResult{}, err
	}
	req = normalizeCallRequest(scope, req)
	if err := req.ToolID.Validate(); err != nil {
		return ToolResult{}, invalidInputError(req.ToolID, "tool id is invalid", err)
	}
	target, err := r.resolveDispatchTarget(ctx, scope, req.ToolID)
	if err != nil {
		if target.descriptor.ID == "" {
			return ToolResult{}, normalizeToolError(req.ToolID, err)
		}
		return ToolResult{}, r.failDispatch(ctx, &target, req, started, err, ToolCallFailed)
	}
	req.Input = normalizeCallInput(req.Input)
	if err := r.ensureDispatchTargetCallable(ctx, &target, req, started); err != nil {
		return ToolResult{}, err
	}
	if err := r.emit(ctx, &target, req, ToolCallStarted, ToolEventData{
		StartedAt: started,
		Input:     req.Input,
	}); err != nil {
		return ToolResult{}, err
	}
	if err := validateCallInput(target.descriptor, req.Input); err != nil {
		return ToolResult{}, r.failDispatch(ctx, &target, req, started, err, ToolCallFailed)
	}
	patchedReq, err := r.runPreCallHook(ctx, &target, req)
	if err != nil {
		return ToolResult{}, r.failDispatch(ctx, &target, req, started, err, ToolCallDenied)
	}
	if err := contextErr(ctx, target.descriptor.ID); err != nil {
		return ToolResult{}, r.failDispatch(ctx, &target, patchedReq, started, err, ToolCallFailed)
	}
	if err := r.requestApproval(ctx, scope, &target, patchedReq); err != nil {
		return ToolResult{}, r.failDispatch(ctx, &target, patchedReq, started, err, ToolCallDenied)
	}
	providerResult, err := target.handle.Call(ctx, patchedReq)
	if err != nil {
		normalized := normalizeBackendError(target.descriptor.ID, err)
		if hookErr := r.runPostErrorHook(ctx, &target, patchedReq, normalized); hookErr != nil {
			normalized = hookErr
		}
		return ToolResult{}, r.failDispatch(ctx, &target, patchedReq, started, normalized, ToolCallFailed)
	}
	limited, err := r.resultLimiter().Apply(ctx, target.descriptor, providerResult)
	if err != nil {
		normalized := normalizeToolError(target.descriptor.ID, err)
		if hookErr := r.runPostErrorHook(ctx, &target, patchedReq, normalized); hookErr != nil {
			normalized = hookErr
		}
		return ToolResult{}, r.failDispatch(ctx, &target, patchedReq, started, normalized, ToolCallFailed)
	}
	limited, err = r.runPostCallHook(ctx, &target, patchedReq, limited)
	if err != nil {
		return ToolResult{}, r.failDispatch(ctx, &target, patchedReq, started, err, ToolCallDenied)
	}
	if err := r.emit(ctx, &target, patchedReq, ToolCallCompleted, ToolEventData{
		StartedAt: started,
		Result:    limited,
	}); err != nil {
		return ToolResult{}, err
	}
	if limited.Truncated {
		if err := r.emit(ctx, &target, patchedReq, ToolResultTruncated, ToolEventData{
			StartedAt: started,
			Result:    limited,
		}); err != nil {
			return ToolResult{}, err
		}
	}
	return limited, nil
}

func normalizeCallRequest(scope Scope, req CallRequest) CallRequest {
	if req.SessionID == "" {
		req.SessionID = scope.SessionID
	}
	if req.WorkspaceID == "" {
		req.WorkspaceID = scope.WorkspaceID
	}
	if req.AgentName == "" {
		req.AgentName = scope.AgentName
	}
	return req
}

func (r *RuntimeRegistry) resolveDispatchTarget(
	ctx context.Context,
	scope Scope,
	id ToolID,
) (dispatchTarget, error) {
	index, err := r.buildIndex(ctx, scope)
	if err != nil {
		return dispatchTarget{}, err
	}
	entry, ok := index.byID[id]
	if !ok {
		return dispatchTarget{}, NewToolError(
			ErrorCodeNotFound,
			id,
			fmt.Sprintf("tool %q not found", id),
			ErrToolNotFound,
			ReasonToolUnknown,
		)
	}
	evaluator, err := r.evaluatorFor(index.ids())
	if err != nil {
		return dispatchTarget{}, err
	}
	handle, availability := r.resolveHandleForDispatch(ctx, scope, entry)
	decision, err := evaluator.Evaluate(ctx, scope, entry.descriptor)
	if err != nil {
		return dispatchTarget{}, err
	}
	decision = applyAvailabilityDecision(decision, availability)
	view := ToolView{
		Descriptor:   cloneDescriptor(entry.descriptor),
		Availability: availability,
		Decision:     decision,
	}
	target := dispatchTarget{descriptor: cloneDescriptor(entry.descriptor), handle: handle, view: view}
	if !decision.Callable || isNilInterface(handle) {
		return target, nil
	}
	if err := ensureHandleMatchesDescriptor(entry.descriptor, handle); err != nil {
		return target, err
	}
	return target, nil
}

func (r *RuntimeRegistry) ensureDispatchTargetCallable(
	ctx context.Context,
	target *dispatchTarget,
	req CallRequest,
	started time.Time,
) error {
	if !target.view.Decision.Callable {
		return r.failDispatch(ctx, target, req, started, denialErrorForView(&target.view), ToolCallDenied)
	}
	if isNilInterface(target.handle) {
		return r.failDispatch(ctx, target, req, started, NewToolError(
			ErrorCodeUnavailable,
			target.descriptor.ID,
			fmt.Sprintf("tool %q is unavailable", target.descriptor.ID),
			ErrToolUnavailable,
			ReasonBackendNotExecutable,
		), ToolCallDenied)
	}
	return nil
}

func (r *RuntimeRegistry) resolveHandleForDispatch(
	ctx context.Context,
	scope Scope,
	entry *registryEntry,
) (Handle, Availability) {
	if len(entry.conflicts) > 0 {
		return nil, Availability{
			Registered:  true,
			Enabled:     true,
			Conflicted:  true,
			ReasonCodes: append([]ReasonCode(nil), entry.conflicts...),
		}
	}
	handle, ok, err := entry.provider.Resolve(ctx, scope, entry.descriptor.ID)
	if err != nil {
		return nil, Availability{
			Registered:  true,
			Enabled:     true,
			ReasonCodes: []ReasonCode{ReasonBackendUnhealthy},
		}
	}
	if !ok || isNilInterface(handle) {
		return nil, Availability{
			Registered:  true,
			Enabled:     true,
			ReasonCodes: []ReasonCode{ReasonBackendNotExecutable},
		}
	}
	availability := handle.Availability(ctx, scope)
	availability.Registered = true
	if err := availability.Validate(); err != nil {
		reason, found := ReasonOf(err)
		if !found {
			reason = ReasonBackendUnhealthy
		}
		return handle, Availability{
			Registered:  true,
			Enabled:     availability.Enabled,
			Available:   availability.Available,
			Authorized:  availability.Authorized,
			Executable:  false,
			Conflicted:  availability.Conflicted,
			ReasonCodes: appendReason(availability.ReasonCodes, reason),
		}
	}
	return handle, availability
}

func ensureHandleMatchesDescriptor(descriptor Descriptor, handle Handle) error {
	if err := ValidateHandle(handle); err != nil {
		return NewToolError(
			ErrorCodeUnavailable,
			descriptor.ID,
			fmt.Sprintf("tool %q handle rejected", descriptor.ID),
			fmt.Errorf("%w: %w", ErrToolUnavailable, err),
			ReasonRuntimeDescriptorMismatch,
		)
	}
	handleDescriptor := handle.Descriptor()
	if handleDescriptor.ID != descriptor.ID || handleDescriptor.Backend.Kind != descriptor.Backend.Kind {
		return NewToolError(
			ErrorCodeUnavailable,
			descriptor.ID,
			fmt.Sprintf("tool %q handle descriptor mismatch", descriptor.ID),
			ErrToolUnavailable,
			ReasonRuntimeDescriptorMismatch,
		)
	}
	return nil
}

func (r *RuntimeRegistry) runPreCallHook(
	ctx context.Context,
	target *dispatchTarget,
	req CallRequest,
) (CallRequest, error) {
	if r.hooks == nil {
		return req, nil
	}
	patched, decision, err := r.hooks.PreCall(ctx, req)
	if err != nil {
		return req, normalizeHookError(target.descriptor.ID, err)
	}
	patched = mergeHookCallRequest(req, patched)
	if patched.ToolID != req.ToolID {
		return req, NewToolError(
			ErrorCodeDenied,
			req.ToolID,
			fmt.Sprintf("tool %q pre-call hook attempted to change tool_id", req.ToolID),
			ErrToolDenied,
			ReasonHookDenied,
		)
	}
	if !decision.Callable {
		reasons := appendUniqueReasons(append([]ReasonCode(nil), decision.ReasonCodes...), ReasonHookDenied)
		return req, NewToolError(
			ErrorCodeDenied,
			req.ToolID,
			fmt.Sprintf("tool %q denied by pre-call hook", req.ToolID),
			ErrToolDenied,
			reasons...,
		)
	}
	if err := validateCallInput(target.descriptor, normalizeCallInput(patched.Input)); err != nil {
		return req, err
	}
	patched.Input = normalizeCallInput(patched.Input)
	return patched, nil
}

func (r *RuntimeRegistry) requestApproval(
	ctx context.Context,
	scope Scope,
	target *dispatchTarget,
	req CallRequest,
) error {
	if target == nil || !target.view.Decision.ApprovalRequired {
		return nil
	}
	if r.approvalBridge == nil {
		return approvalBridgeError(
			target.descriptor.ID,
			"tool approval channel is unavailable",
			ErrToolApprovalRequired,
			ReasonApprovalUnreachable,
		)
	}
	view := cloneToolView(&target.view)
	if err := r.approvalBridge.RequestToolApproval(ctx, scope, req, &view); err != nil {
		return normalizeApprovalBridgeError(target.descriptor.ID, err)
	}
	return nil
}

func normalizeApprovalBridgeError(id ToolID, err error) error {
	if err == nil {
		return nil
	}
	var toolErr *ToolError
	if errors.As(err, &toolErr) {
		return normalizeToolError(id, toolErr)
	}
	switch {
	case errors.Is(err, context.Canceled):
		return approvalBridgeError(id, "tool approval was canceled", ErrToolApprovalRequired, ReasonApprovalCanceled)
	case errors.Is(err, context.DeadlineExceeded):
		return approvalBridgeError(id, "tool approval timed out", ErrToolApprovalRequired, ReasonApprovalTimedOut)
	case errors.Is(err, ErrToolApprovalRequired):
		return approvalBridgeError(id, "tool approval was not granted", err)
	default:
		return approvalBridgeError(id, "tool approval failed", fmt.Errorf("%w: %w", ErrToolApprovalRequired, err))
	}
}

func approvalBridgeError(id ToolID, message string, err error, reasons ...ReasonCode) *ToolError {
	allReasons := appendUniqueReasons([]ReasonCode{ReasonApprovalRequired}, reasons...)
	return NewToolError(
		ErrorCodeApprovalRequired,
		id,
		fmt.Sprintf("%s for %q", message, id),
		err,
		allReasons...,
	)
}

func mergeHookCallRequest(original CallRequest, patched CallRequest) CallRequest {
	if patched.ToolID == "" {
		patched.ToolID = original.ToolID
	}
	if patched.ToolCallID == "" {
		patched.ToolCallID = original.ToolCallID
	}
	if patched.TurnID == "" {
		patched.TurnID = original.TurnID
	}
	if patched.SessionID == "" {
		patched.SessionID = original.SessionID
	}
	if patched.WorkspaceID == "" {
		patched.WorkspaceID = original.WorkspaceID
	}
	if patched.AgentName == "" {
		patched.AgentName = original.AgentName
	}
	if patched.CorrelationID == "" {
		patched.CorrelationID = original.CorrelationID
	}
	if len(patched.Input) == 0 {
		patched.Input = original.Input
	}
	if len(patched.SensitiveInputFields) == 0 {
		patched.SensitiveInputFields = append([]string(nil), original.SensitiveInputFields...)
	}
	if patched.ApprovalToken == "" {
		patched.ApprovalToken = original.ApprovalToken
	}
	return patched
}

func (r *RuntimeRegistry) runPostCallHook(
	ctx context.Context,
	target *dispatchTarget,
	req CallRequest,
	result ToolResult,
) (ToolResult, error) {
	if r.hooks == nil {
		return result, nil
	}
	patched, err := r.hooks.PostCall(ctx, req, result)
	if err != nil {
		return result, normalizeHookError(target.descriptor.ID, err)
	}
	limited, err := r.resultLimiter().Apply(ctx, target.descriptor, patched)
	if err != nil {
		return result, err
	}
	return limited, nil
}

func (r *RuntimeRegistry) runPostErrorHook(
	ctx context.Context,
	target *dispatchTarget,
	req CallRequest,
	callErr error,
) error {
	if r.hooks == nil {
		return nil
	}
	if err := r.hooks.PostError(ctx, req, callErr); err != nil {
		return normalizeHookError(target.descriptor.ID, err)
	}
	return nil
}

func (r *RuntimeRegistry) failDispatch(
	ctx context.Context,
	target *dispatchTarget,
	req CallRequest,
	started time.Time,
	err error,
	kind ToolCallEventKind,
) error {
	normalized := normalizeToolError(target.descriptor.ID, err)
	if emitErr := r.emit(ctx, target, req, kind, ToolEventData{
		StartedAt: started,
		Err:       normalized,
	}); emitErr != nil {
		return emitErr
	}
	return normalized
}

func (r *RuntimeRegistry) resultLimiter() ResultLimiter {
	if r.limiter != nil {
		return r.limiter
	}
	return NewResultLimiter(r.defaultMaxResultBytes, r.sensitiveFields...)
}

func contextErr(ctx context.Context, id ToolID) error {
	if ctx == nil {
		return NewToolError(
			ErrorCodeCanceled,
			id,
			"tool call context is required",
			ErrToolCanceled,
			ReasonCallCanceled,
		)
	}
	switch err := ctx.Err(); {
	case errors.Is(err, context.Canceled):
		return NewToolError(ErrorCodeCanceled, id, "tool call canceled", ErrToolCanceled, ReasonCallCanceled)
	case errors.Is(err, context.DeadlineExceeded):
		return NewToolError(ErrorCodeTimedOut, id, "tool call timed out", ErrToolTimedOut, ReasonCallTimedOut)
	default:
		return nil
	}
}

func normalizeBackendError(id ToolID, err error) error {
	if contextError := contextErrFromError(id, err); contextError != nil {
		return contextError
	}
	return normalizeToolError(id, NewToolError(
		ErrorCodeBackendFailed,
		id,
		fmt.Sprintf("tool %q backend failed", id),
		fmt.Errorf("%w: %w", ErrToolBackendFailed, err),
		ReasonBackendUnhealthy,
	))
}

func normalizeHookError(id ToolID, err error) error {
	if contextError := contextErrFromError(id, err); contextError != nil {
		return contextError
	}
	return NewToolError(
		ErrorCodeDenied,
		id,
		fmt.Sprintf("tool %q denied by hook", id),
		fmt.Errorf("%w: %w", ErrToolDenied, err),
		ReasonHookDenied,
	)
}

func invalidInputError(id ToolID, message string, err error) error {
	return NewToolError(
		ErrorCodeInvalidInput,
		id,
		message,
		fmt.Errorf("%w: %w", ErrToolInvalidInput, err),
		ReasonSchemaInvalid,
	)
}

func normalizeToolError(id ToolID, err error) error {
	if err == nil {
		return nil
	}
	var toolErr *ToolError
	if errors.As(err, &toolErr) {
		if toolErr.ToolID == "" && id != "" {
			cloned := *toolErr
			cloned.ToolID = id
			return &cloned
		}
		return toolErr
	}
	if contextError := contextErrFromError(id, err); contextError != nil {
		return contextError
	}
	return NewToolError(
		ErrorCodeBackendFailed,
		id,
		fmt.Sprintf("tool %q failed", id),
		fmt.Errorf("%w: %w", ErrToolBackendFailed, err),
		ReasonBackendUnhealthy,
	)
}

func contextErrFromError(id ToolID, err error) error {
	switch {
	case errors.Is(err, context.Canceled), errors.Is(err, ErrToolCanceled):
		return NewToolError(ErrorCodeCanceled, id, "tool call canceled", ErrToolCanceled, ReasonCallCanceled)
	case errors.Is(err, context.DeadlineExceeded), errors.Is(err, ErrToolTimedOut):
		return NewToolError(ErrorCodeTimedOut, id, "tool call timed out", ErrToolTimedOut, ReasonCallTimedOut)
	default:
		return nil
	}
}

// ToolEventData carries per-outcome event details.
type ToolEventData struct {
	StartedAt time.Time
	Input     json.RawMessage
	Result    ToolResult
	Err       error
}

func (r *RuntimeRegistry) emit(
	ctx context.Context,
	target *dispatchTarget,
	req CallRequest,
	kind ToolCallEventKind,
	data ToolEventData,
) error {
	if r.events == nil {
		return nil
	}
	event := buildToolCallEvent(target, req, kind, data)
	if err := r.events.EmitToolEvent(ctx, event); err != nil {
		return NewToolError(
			ErrorCodeBackendFailed,
			target.descriptor.ID,
			fmt.Sprintf("tool %q observability emit failed", target.descriptor.ID),
			fmt.Errorf("%w: %w", ErrToolBackendFailed, err),
			ReasonBackendUnhealthy,
		)
	}
	return nil
}

func buildToolCallEvent(
	target *dispatchTarget,
	req CallRequest,
	kind ToolCallEventKind,
	data ToolEventData,
) ToolCallEvent {
	descriptor := target.descriptor
	event := ToolCallEvent{
		Kind:          kind,
		ToolID:        descriptor.ID,
		DisplayTitle:  descriptor.DisplayTitle,
		SourceKind:    descriptor.Source.Kind,
		SourceOwner:   descriptor.Source.Owner,
		WorkspaceID:   req.WorkspaceID,
		SessionID:     req.SessionID,
		AgentName:     req.AgentName,
		Risk:          descriptor.Risk,
		ReadOnly:      descriptor.ReadOnly,
		Destructive:   descriptor.Destructive,
		OpenWorld:     descriptor.OpenWorld,
		ApprovalMode:  target.view.Decision.SystemPermissionMode,
		Decision:      target.view.Decision.RegistryPolicyResult,
		ReasonCodes:   append([]ReasonCode(nil), target.view.Decision.ReasonCodes...),
		CorrelationID: req.CorrelationID,
		InputDigest:   digestRaw(redactInputForEvents(req.Input, req.SensitiveInputFields)),
	}
	if !data.StartedAt.IsZero() {
		event.DurationMS = time.Since(data.StartedAt).Milliseconds()
	}
	event.RedactedInputFields = redactedInputFields(req.Input, req.SensitiveInputFields)
	if data.Result.Bytes > 0 || data.Result.Truncated {
		event.ResultBytes = data.Result.Bytes
		event.Truncated = data.Result.Truncated
		event.ResultDigest = digestToolResult(data.Result)
		event.ResultRedactionPaths = resultRedactionPaths(data.Result.Redactions)
	}
	if data.Err != nil {
		var toolErr *ToolError
		if errors.As(data.Err, &toolErr) {
			event.ErrorCode = toolErr.Code
			event.ReasonCodes = appendUniqueReasons(event.ReasonCodes, toolErr.ReasonCodes...)
		}
	}
	return event
}

func redactInputForEvents(input json.RawMessage, fields []string) json.RawMessage {
	redacted, _, err := redactRawJSON(normalizeCallInput(input), "$.input", normalizeSensitiveFields(fields), nil)
	if err != nil {
		return json.RawMessage(`{"invalid":true}`)
	}
	return redacted
}

func redactedInputFields(input json.RawMessage, fields []string) []string {
	_, redactions, err := redactRawJSON(normalizeCallInput(input), "$.input", normalizeSensitiveFields(fields), nil)
	if err != nil {
		return nil
	}
	values := make([]string, 0, len(redactions))
	for _, redaction := range redactions {
		values = append(values, redaction.Path)
	}
	return values
}

func digestToolResult(result ToolResult) string {
	data, err := json.Marshal(result)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func resultRedactionPaths(redactions []Redaction) []string {
	paths := make([]string, 0, len(redactions))
	for _, redaction := range redactions {
		if redaction.Path != "" {
			paths = append(paths, redaction.Path)
		}
	}
	return paths
}

func appendUniqueReasons(existing []ReasonCode, incoming ...ReasonCode) []ReasonCode {
	for _, reason := range incoming {
		if reason != "" && !slices.Contains(existing, reason) {
			existing = append(existing, reason)
		}
	}
	return existing
}
