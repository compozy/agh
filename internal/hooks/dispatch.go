package hooks

import (
	"context"
	"errors"
	"fmt"
	"time"
)

type dispatchConfig[P any, R any] struct {
	match   matcherFunc[P]
	apply   func(P, R) P
	denied  denyDetector[R]
	denyErr func(P) error
	guard   patchGuard[P, R]
}

// DispatchSessionPreCreate runs the session.pre_create hook pipeline.
func (h *Hooks) DispatchSessionPreCreate(ctx context.Context, payload SessionPreCreatePayload) (SessionPreCreatePayload, error) {
	return executeDispatch(
		h,
		ctx,
		HookSessionPreCreate,
		payload,
		dispatchConfig[SessionPreCreatePayload, SessionCreatePatch]{
			match:  matchSessionPreCreate,
			apply:  applySessionCreatePatch,
			denied: sessionCreatePatchDenied,
			denyErr: func(SessionPreCreatePayload) error {
				return fmt.Errorf("hooks: event %q denied", HookSessionPreCreate)
			},
		},
	)
}

// DispatchSessionPostCreate runs the session.post_create hook pipeline.
func (h *Hooks) DispatchSessionPostCreate(ctx context.Context, payload SessionPostCreatePayload) (SessionPostCreatePayload, error) {
	return executeDispatch(
		h,
		ctx,
		HookSessionPostCreate,
		payload,
		dispatchConfig[SessionPostCreatePayload, SessionPostCreatePatch]{
			match:  matchSessionLifecycle,
			apply:  applySessionLifecyclePatch,
			denied: sessionCreatePatchDenied,
		},
	)
}

// DispatchSessionPreResume runs the session.pre_resume hook pipeline.
func (h *Hooks) DispatchSessionPreResume(ctx context.Context, payload SessionPreResumePayload) (SessionPreResumePayload, error) {
	return executeDispatch(
		h,
		ctx,
		HookSessionPreResume,
		payload,
		dispatchConfig[SessionPreResumePayload, SessionPreResumePatch]{
			match:  matchSessionLifecycle,
			apply:  applySessionLifecyclePatch,
			denied: sessionCreatePatchDenied,
			denyErr: func(SessionPreResumePayload) error {
				return fmt.Errorf("hooks: event %q denied", HookSessionPreResume)
			},
		},
	)
}

// DispatchSessionPostResume runs the session.post_resume hook pipeline.
func (h *Hooks) DispatchSessionPostResume(ctx context.Context, payload SessionPostResumePayload) (SessionPostResumePayload, error) {
	return executeDispatch(
		h,
		ctx,
		HookSessionPostResume,
		payload,
		dispatchConfig[SessionPostResumePayload, SessionPostResumePatch]{
			match:  matchSessionLifecycle,
			apply:  applySessionLifecyclePatch,
			denied: sessionCreatePatchDenied,
		},
	)
}

// DispatchSessionPreStop runs the session.pre_stop hook pipeline.
func (h *Hooks) DispatchSessionPreStop(ctx context.Context, payload SessionPreStopPayload) (SessionPreStopPayload, error) {
	return executeDispatch(
		h,
		ctx,
		HookSessionPreStop,
		payload,
		dispatchConfig[SessionPreStopPayload, SessionPreStopPatch]{
			match:  matchSessionLifecycle,
			apply:  applySessionLifecyclePatch,
			denied: sessionCreatePatchDenied,
			denyErr: func(SessionPreStopPayload) error {
				return fmt.Errorf("hooks: event %q denied", HookSessionPreStop)
			},
		},
	)
}

// DispatchSessionPostStop runs the session.post_stop hook pipeline.
func (h *Hooks) DispatchSessionPostStop(ctx context.Context, payload SessionPostStopPayload) (SessionPostStopPayload, error) {
	return executeDispatch(
		h,
		ctx,
		HookSessionPostStop,
		payload,
		dispatchConfig[SessionPostStopPayload, SessionPostStopPatch]{
			match:  matchSessionLifecycle,
			apply:  applySessionLifecyclePatch,
			denied: sessionCreatePatchDenied,
		},
	)
}

// DispatchInputPreSubmit runs the input.pre_submit hook pipeline.
func (h *Hooks) DispatchInputPreSubmit(ctx context.Context, payload InputPreSubmitPayload) (InputPreSubmitPayload, error) {
	return executeDispatch(
		h,
		ctx,
		HookInputPreSubmit,
		payload,
		dispatchConfig[InputPreSubmitPayload, InputPreSubmitPatch]{
			match:  matchInputPreSubmit,
			apply:  applyInputPreSubmitPatch,
			denied: inputPreSubmitPatchDenied,
			denyErr: func(InputPreSubmitPayload) error {
				return fmt.Errorf("hooks: event %q denied", HookInputPreSubmit)
			},
		},
	)
}

// DispatchPromptPostAssemble runs the prompt.post_assemble hook pipeline.
func (h *Hooks) DispatchPromptPostAssemble(ctx context.Context, payload PromptPayload) (PromptPayload, error) {
	return executeDispatch(
		h,
		ctx,
		HookPromptPostAssemble,
		payload,
		dispatchConfig[PromptPayload, PromptPatch]{
			match:  matchPrompt,
			apply:  applyPromptPatch,
			denied: promptPatchDenied,
			denyErr: func(PromptPayload) error {
				return fmt.Errorf("hooks: event %q denied", HookPromptPostAssemble)
			},
		},
	)
}

// DispatchEventPreRecord runs the event.pre_record hook dispatch.
func (h *Hooks) DispatchEventPreRecord(ctx context.Context, payload EventPreRecordPayload) (EventPreRecordPayload, error) {
	return executeDispatch(
		h,
		ctx,
		HookEventPreRecord,
		payload,
		dispatchConfig[EventPreRecordPayload, EventPreRecordPatch]{
			match: matchEventRecord,
			apply: applyNoop[EventPreRecordPayload, EventPreRecordPatch],
		},
	)
}

// DispatchEventPostRecord runs the event.post_record hook dispatch.
func (h *Hooks) DispatchEventPostRecord(ctx context.Context, payload EventPostRecordPayload) (EventPostRecordPayload, error) {
	return executeDispatch(
		h,
		ctx,
		HookEventPostRecord,
		payload,
		dispatchConfig[EventPostRecordPayload, EventPostRecordPatch]{
			match: matchEventRecord,
			apply: applyNoop[EventPostRecordPayload, EventPostRecordPatch],
		},
	)
}

// DispatchAgentPreStart runs the agent.pre_start hook pipeline.
func (h *Hooks) DispatchAgentPreStart(ctx context.Context, payload AgentPreStartPayload) (AgentPreStartPayload, error) {
	return executeDispatch(
		h,
		ctx,
		HookAgentPreStart,
		payload,
		dispatchConfig[AgentPreStartPayload, AgentStartPatch]{
			match:  matchAgentPreStart,
			apply:  applyAgentStartPatch,
			denied: agentStartPatchDenied,
			denyErr: func(AgentPreStartPayload) error {
				return fmt.Errorf("hooks: event %q denied", HookAgentPreStart)
			},
		},
	)
}

// DispatchAgentSpawned runs the agent.spawned hook pipeline.
func (h *Hooks) DispatchAgentSpawned(ctx context.Context, payload AgentSpawnedPayload) (AgentSpawnedPayload, error) {
	return executeDispatch(
		h,
		ctx,
		HookAgentSpawned,
		payload,
		dispatchConfig[AgentSpawnedPayload, AgentSpawnedPatch]{
			match: matchAgentLifecycle,
			apply: applyNoop[AgentSpawnedPayload, AgentSpawnedPatch],
		},
	)
}

// DispatchAgentCrashed runs the agent.crashed hook pipeline.
func (h *Hooks) DispatchAgentCrashed(ctx context.Context, payload AgentCrashedPayload) (AgentCrashedPayload, error) {
	return executeDispatch(
		h,
		ctx,
		HookAgentCrashed,
		payload,
		dispatchConfig[AgentCrashedPayload, AgentCrashedPatch]{
			match: matchAgentLifecycle,
			apply: applyNoop[AgentCrashedPayload, AgentCrashedPatch],
		},
	)
}

// DispatchAgentStopped runs the agent.stopped hook pipeline.
func (h *Hooks) DispatchAgentStopped(ctx context.Context, payload AgentStoppedPayload) (AgentStoppedPayload, error) {
	return executeDispatch(
		h,
		ctx,
		HookAgentStopped,
		payload,
		dispatchConfig[AgentStoppedPayload, AgentStoppedPatch]{
			match: matchAgentLifecycle,
			apply: applyNoop[AgentStoppedPayload, AgentStoppedPatch],
		},
	)
}

// DispatchTurnStart runs the turn.start hook pipeline.
func (h *Hooks) DispatchTurnStart(ctx context.Context, payload TurnStartPayload) (TurnStartPayload, error) {
	return executeDispatch(
		h,
		ctx,
		HookTurnStart,
		payload,
		dispatchConfig[TurnStartPayload, TurnStartPatch]{
			match:  matchTurn,
			apply:  applyNoop[TurnStartPayload, TurnStartPatch],
			denied: turnPatchDenied,
		},
	)
}

// DispatchTurnEnd runs the turn.end hook pipeline.
func (h *Hooks) DispatchTurnEnd(ctx context.Context, payload TurnEndPayload) (TurnEndPayload, error) {
	return executeDispatch(
		h,
		ctx,
		HookTurnEnd,
		payload,
		dispatchConfig[TurnEndPayload, TurnEndPatch]{
			match:  matchTurn,
			apply:  applyNoop[TurnEndPayload, TurnEndPatch],
			denied: turnPatchDenied,
		},
	)
}

// DispatchMessageStart runs the message.start hook pipeline.
func (h *Hooks) DispatchMessageStart(ctx context.Context, payload MessageStartPayload) (MessageStartPayload, error) {
	return executeDispatch(
		h,
		ctx,
		HookMessageStart,
		payload,
		dispatchConfig[MessageStartPayload, MessageStartPatch]{
			match:  matchMessage,
			apply:  applyMessagePatch,
			denied: messagePatchDenied,
		},
	)
}

// DispatchMessageDelta runs the message.delta hook dispatch.
func (h *Hooks) DispatchMessageDelta(ctx context.Context, payload MessageDeltaPayload) (MessageDeltaPayload, error) {
	return executeDispatch(
		h,
		ctx,
		HookMessageDelta,
		payload,
		dispatchConfig[MessageDeltaPayload, MessageDeltaPatch]{
			match:  matchMessage,
			apply:  applyMessagePatch,
			denied: messagePatchDenied,
		},
	)
}

// DispatchMessageEnd runs the message.end hook pipeline.
func (h *Hooks) DispatchMessageEnd(ctx context.Context, payload MessageEndPayload) (MessageEndPayload, error) {
	return executeDispatch(
		h,
		ctx,
		HookMessageEnd,
		payload,
		dispatchConfig[MessageEndPayload, MessageEndPatch]{
			match:  matchMessage,
			apply:  applyMessagePatch,
			denied: messagePatchDenied,
		},
	)
}

// DispatchToolPreCall runs the tool.pre_call hook pipeline.
func (h *Hooks) DispatchToolPreCall(ctx context.Context, payload ToolPreCallPayload) (ToolPreCallPayload, error) {
	return executeDispatch(
		h,
		ctx,
		HookToolPreCall,
		payload,
		dispatchConfig[ToolPreCallPayload, ToolCallPatch]{
			match:  matchToolPreCall,
			apply:  applyToolCallPatch,
			denied: toolCallPatchDenied,
		},
	)
}

// DispatchToolPostCall runs the tool.post_call hook pipeline.
func (h *Hooks) DispatchToolPostCall(ctx context.Context, payload ToolPostCallPayload) (ToolPostCallPayload, error) {
	return executeDispatch(
		h,
		ctx,
		HookToolPostCall,
		payload,
		dispatchConfig[ToolPostCallPayload, ToolResultPatch]{
			match:  matchToolPostCall,
			apply:  applyToolResultPatch,
			denied: toolResultPatchDenied,
		},
	)
}

// DispatchToolPostError runs the tool.post_error hook pipeline.
func (h *Hooks) DispatchToolPostError(ctx context.Context, payload ToolPostErrorPayload) (ToolPostErrorPayload, error) {
	return executeDispatch(
		h,
		ctx,
		HookToolPostError,
		payload,
		dispatchConfig[ToolPostErrorPayload, ToolPostErrorPatch]{
			match:  matchToolPostError,
			apply:  applyToolPostErrorPatch,
			denied: toolResultPatchDenied,
		},
	)
}

// DispatchPermissionRequest runs the permission.request hook pipeline.
func (h *Hooks) DispatchPermissionRequest(ctx context.Context, payload PermissionRequestPayload) (PermissionRequestPayload, error) {
	return executeDispatch(
		h,
		ctx,
		HookPermissionRequest,
		payload,
		dispatchConfig[PermissionRequestPayload, PermissionRequestPatch]{
			match:  matchPermissionRequest,
			apply:  mergePermissionRequestPatch,
			denied: permissionPatchDenies,
			guard:  newPermissionRequestGuard(h.logger, h.metrics),
		},
	)
}

// DispatchPermissionResolved runs the permission.resolved hook dispatch.
func (h *Hooks) DispatchPermissionResolved(ctx context.Context, payload PermissionResolvedPayload) (PermissionResolvedPayload, error) {
	return executeDispatch(
		h,
		ctx,
		HookPermissionResolved,
		payload,
		dispatchConfig[PermissionResolvedPayload, PermissionResolvedPatch]{
			match: matchPermissionResolution,
			apply: applyNoop[PermissionResolvedPayload, PermissionResolvedPatch],
		},
	)
}

// DispatchPermissionDenied runs the permission.denied hook dispatch.
func (h *Hooks) DispatchPermissionDenied(ctx context.Context, payload PermissionDeniedPayload) (PermissionDeniedPayload, error) {
	return executeDispatch(
		h,
		ctx,
		HookPermissionDenied,
		payload,
		dispatchConfig[PermissionDeniedPayload, PermissionDeniedPatch]{
			match: matchPermissionResolution,
			apply: applyNoop[PermissionDeniedPayload, PermissionDeniedPatch],
		},
	)
}

// DispatchContextPreCompact runs the context.pre_compact hook pipeline.
func (h *Hooks) DispatchContextPreCompact(ctx context.Context, payload ContextPreCompactPayload) (ContextPreCompactPayload, error) {
	return executeDispatch(
		h,
		ctx,
		HookContextPreCompact,
		payload,
		dispatchConfig[ContextPreCompactPayload, ContextPreCompactPatch]{
			match:  matchContextCompact,
			apply:  applyContextCompactionPatch,
			denied: contextCompactionPatchDenied,
		},
	)
}

// DispatchContextPostCompact runs the context.post_compact hook pipeline.
func (h *Hooks) DispatchContextPostCompact(ctx context.Context, payload ContextPostCompactPayload) (ContextPostCompactPayload, error) {
	return executeDispatch(
		h,
		ctx,
		HookContextPostCompact,
		payload,
		dispatchConfig[ContextPostCompactPayload, ContextPostCompactPatch]{
			match:  matchContextCompact,
			apply:  applyContextCompactionPatch,
			denied: contextCompactionPatchDenied,
		},
	)
}

func executeDispatch[P any, R any](
	h *Hooks,
	ctx context.Context,
	event HookEvent,
	payload P,
	cfg dispatchConfig[P, R],
) (P, error) {
	if h == nil {
		return payload, errors.New("hooks: dispatcher is nil")
	}
	if ctx == nil {
		return payload, errors.New("hooks: dispatch context is nil")
	}

	snapshot, err := h.hookSnapshot(event)
	if err != nil {
		return payload, err
	}

	syncHooks, asyncHooks := selectMatchingHooks(snapshot, payload, cfg.match)
	if len(syncHooks) == 0 && len(asyncHooks) == 0 {
		return payload, nil
	}

	dispatchDepth := currentDispatchDepth(ctx) + 1
	dispatchStarted := time.Now()
	h.logger.Info(
		"hook.dispatch.started",
		"event", event.String(),
		"dispatch_depth", dispatchDepth,
		"sync_hooks", len(syncHooks),
		"async_hooks", len(asyncHooks),
	)

	result := payload
	var dispatchErr error
	pipe := pipeline[P, R]{
		event:        event,
		hooksRuntime: h,
		hooks:        func(P) []*ResolvedHook { return syncHooks },
		apply:        cfg.apply,
		encode:       encodeJSON[P],
		decode:       decodeJSON[R],
		denied:       cfg.denied,
		guard:        cfg.guard,
		enter:        h.enterDispatch,
	}
	var report dispatchReport
	if len(syncHooks) > 0 {
		result, report, dispatchErr = pipe.executeWithDisposition(ctx, payload)
		if dispatchErr == nil && report.Denied && cfg.denyErr != nil {
			dispatchErr = cfg.denyErr(result)
		}
	}

	if dispatchErr == nil && !report.Denied && len(asyncHooks) > 0 {
		submitAsyncHooks(h, ctx, result, asyncHooks, pipe)
	}

	pipelineDuration := time.Since(dispatchStarted)
	h.metrics.observePipeline(event, pipelineDuration)
	switch {
	case report.Denied:
		h.logger.Warn(
			"hook.dispatch.blocked",
			"event", event.String(),
			"dispatch_depth", dispatchDepth,
			"deny_source", report.DenySource,
			"pipeline_trace", traceStrings(report.Trace),
		)
	case dispatchErr != nil:
		h.logger.Warn(
			"hook.dispatch.failed",
			"event", event.String(),
			"dispatch_depth", dispatchDepth,
			"error", dispatchErr,
			"failed_hook", report.FailedHook,
			"required", report.FailedRequired,
			"pipeline_trace", traceStrings(report.Trace),
		)
	default:
		h.logger.Info(
			"hook.dispatch.completed",
			"event", event.String(),
			"dispatch_depth", dispatchDepth,
			"duration_ms", pipelineDuration.Milliseconds(),
			"pipeline_trace", traceStrings(report.Trace),
			"sync_hooks", len(syncHooks),
			"async_hooks", len(asyncHooks),
		)
	}

	return result, dispatchErr
}

func applyNoop[P any, R any](payload P, _ R) P {
	return payload
}

func applySessionContextPatch(payload SessionContext, patch SessionCreatePatch) SessionContext {
	if patch.SessionName != nil {
		payload.SessionName = *patch.SessionName
	}
	if patch.SessionType != nil {
		payload.SessionType = *patch.SessionType
	}
	if patch.AgentName != nil {
		payload.AgentName = *patch.AgentName
	}
	if patch.WorkspaceID != nil {
		payload.WorkspaceID = *patch.WorkspaceID
	}
	if patch.Workspace != nil {
		payload.Workspace = *patch.Workspace
	}
	return payload
}

func applySessionCreatePatch(payload SessionPreCreatePayload, patch SessionCreatePatch) SessionPreCreatePayload {
	payload.SessionContext = applySessionContextPatch(payload.SessionContext, patch)
	return payload
}

func applySessionLifecyclePatch(payload SessionLifecyclePayload, patch SessionCreatePatch) SessionLifecyclePayload {
	payload.SessionContext = applySessionContextPatch(payload.SessionContext, patch)
	return payload
}

func applyInputPreSubmitPatch(payload InputPreSubmitPayload, patch InputPreSubmitPatch) InputPreSubmitPayload {
	if patch.Message != nil {
		payload.Message = *patch.Message
	}
	if patch.ContextBlocks != nil {
		payload.ContextBlocks = cloneContextBlocks(patch.ContextBlocks)
	}
	return payload
}

func applyPromptPatch(payload PromptPayload, patch PromptPatch) PromptPayload {
	if patch.Prompt != nil {
		payload.Prompt = *patch.Prompt
	}
	if patch.ContextBlocks != nil {
		payload.ContextBlocks = cloneContextBlocks(patch.ContextBlocks)
	}
	return payload
}

func applyAgentStartPatch(payload AgentPreStartPayload, patch AgentStartPatch) AgentPreStartPayload {
	if patch.Command != nil {
		payload.Command = *patch.Command
	}
	if patch.Args != nil {
		payload.Args = append([]string(nil), patch.Args...)
	}
	if patch.Cwd != nil {
		payload.Cwd = *patch.Cwd
	}
	return payload
}

func applyMessagePatch(payload MessagePayload, patch MessagePatch) MessagePayload {
	if patch.Role != nil {
		payload.Role = *patch.Role
	}
	if patch.DeltaType != nil {
		payload.DeltaType = *patch.DeltaType
	}
	if patch.Text != nil {
		payload.Text = *patch.Text
	}
	return payload
}

func applyToolCallPatch(payload ToolPreCallPayload, patch ToolCallPatch) ToolPreCallPayload {
	if patch.ToolName != nil {
		payload.ToolName = *patch.ToolName
	}
	if patch.ToolNamespace != nil {
		payload.ToolNamespace = *patch.ToolNamespace
	}
	if patch.ReadOnly != nil {
		payload.ReadOnly = *patch.ReadOnly
	}
	if patch.ToolInput != nil {
		payload.ToolInput = cloneRawMessage(patch.ToolInput)
	}
	return payload
}

func applyToolResultPatch(payload ToolPostCallPayload, patch ToolResultPatch) ToolPostCallPayload {
	if patch.Title != nil {
		payload.Title = *patch.Title
	}
	if patch.ToolResult != nil {
		payload.ToolResult = cloneRawMessage(patch.ToolResult)
	}
	return payload
}

func applyToolPostErrorPatch(payload ToolPostErrorPayload, patch ToolPostErrorPatch) ToolPostErrorPayload {
	if patch.Title != nil {
		payload.Title = *patch.Title
	}
	if patch.Error != nil {
		payload.Error = *patch.Error
	}
	return payload
}

func mergePermissionRequestPatch(payload PermissionRequestPayload, patch PermissionRequestPatch) PermissionRequestPayload {
	if patch.Decision != nil {
		payload.Decision = *patch.Decision
	}
	if patch.Deny {
		payload.Decision = "deny"
	}
	if patch.DecisionClass != nil {
		payload.DecisionClass = *patch.DecisionClass
	}
	return payload
}

func applyContextCompactionPatch(payload ContextCompactPayload, patch ContextCompactionPatch) ContextCompactPayload {
	if patch.Reason != nil {
		payload.Reason = *patch.Reason
	}
	if patch.Strategy != nil {
		payload.Strategy = *patch.Strategy
	}
	if patch.ContextBlocks != nil {
		payload.ContextBlocks = cloneContextBlocks(patch.ContextBlocks)
	}
	return payload
}

func cloneContextBlocks(blocks []ContextBlock) []ContextBlock {
	if blocks == nil {
		return nil
	}

	cloned := make([]ContextBlock, 0, len(blocks))
	for _, block := range blocks {
		cloned = append(cloned, ContextBlock{
			Kind:     block.Kind,
			Text:     block.Text,
			Metadata: cloneStringMap(block.Metadata),
		})
	}
	return cloned
}

func cloneRawMessage(payload []byte) []byte {
	if payload == nil {
		return nil
	}

	return append([]byte(nil), payload...)
}

func sessionCreatePatchDenied(patch SessionCreatePatch) bool {
	return patch.Deny
}

func inputPreSubmitPatchDenied(patch InputPreSubmitPatch) bool {
	return patch.Deny
}

func promptPatchDenied(patch PromptPatch) bool {
	return patch.Deny
}

func agentStartPatchDenied(patch AgentStartPatch) bool {
	return patch.Deny
}

func turnPatchDenied(patch TurnPatch) bool {
	return patch.Deny
}

func messagePatchDenied(patch MessagePatch) bool {
	return patch.Deny
}

func toolCallPatchDenied(patch ToolCallPatch) bool {
	return patch.Deny
}

func toolResultPatchDenied(patch ToolResultPatch) bool {
	return patch.Deny
}

func contextCompactionPatchDenied(patch ContextCompactionPatch) bool {
	return patch.Deny
}
