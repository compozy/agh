package hooks

import (
	"context"
	"errors"
	"fmt"
	"strings"
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
func (h *Hooks) DispatchSessionPreCreate(
	ctx context.Context,
	payload SessionPreCreatePayload,
) (SessionPreCreatePayload, error) {
	return executeDispatch(
		ctx,
		h,
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
func (h *Hooks) DispatchSessionPostCreate(
	ctx context.Context,
	payload SessionPostCreatePayload,
) (SessionPostCreatePayload, error) {
	return executeDispatch(
		ctx,
		h,
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
func (h *Hooks) DispatchSessionPreResume(
	ctx context.Context,
	payload SessionPreResumePayload,
) (SessionPreResumePayload, error) {
	return executeDispatch(
		ctx,
		h,
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
func (h *Hooks) DispatchSessionPostResume(
	ctx context.Context,
	payload SessionPostResumePayload,
) (SessionPostResumePayload, error) {
	return executeDispatch(
		ctx,
		h,
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
func (h *Hooks) DispatchSessionPreStop(
	ctx context.Context,
	payload SessionPreStopPayload,
) (SessionPreStopPayload, error) {
	return executeDispatch(
		ctx,
		h,
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
func (h *Hooks) DispatchSessionPostStop(
	ctx context.Context,
	payload SessionPostStopPayload,
) (SessionPostStopPayload, error) {
	return executeDispatch(
		ctx,
		h,
		HookSessionPostStop,
		payload,
		dispatchConfig[SessionPostStopPayload, SessionPostStopPatch]{
			match:  matchSessionLifecycle,
			apply:  applySessionLifecyclePatch,
			denied: sessionCreatePatchDenied,
		},
	)
}

// DispatchSandboxPrepare runs the sandbox.prepare hook pipeline.
func (h *Hooks) DispatchSandboxPrepare(
	ctx context.Context,
	payload SandboxPreparePayload,
) (SandboxPreparePayload, error) {
	return executeDispatch(
		ctx,
		h,
		HookSandboxPrepare,
		payload,
		dispatchConfig[SandboxPreparePayload, SandboxPreparePatch]{
			match:  matchSandboxPrepare,
			apply:  applySandboxPreparePatch,
			denied: sandboxPreparePatchDenied,
			denyErr: func(SandboxPreparePayload) error {
				return fmt.Errorf("hooks: event %q denied", HookSandboxPrepare)
			},
		},
	)
}

// DispatchSandboxReady runs the sandbox.ready hook dispatch.
func (h *Hooks) DispatchSandboxReady(
	ctx context.Context,
	payload SandboxReadyPayload,
) (SandboxReadyPayload, error) {
	return executeDispatch(
		ctx,
		h,
		HookSandboxReady,
		payload,
		dispatchConfig[SandboxReadyPayload, SandboxReadyPatch]{
			match: matchSandboxReady,
			apply: applyNoop[SandboxReadyPayload, SandboxReadyPatch],
		},
	)
}

// DispatchSandboxSyncBefore runs the sandbox.sync.before hook pipeline.
func (h *Hooks) DispatchSandboxSyncBefore(
	ctx context.Context,
	payload SandboxSyncBeforePayload,
) (SandboxSyncBeforePayload, error) {
	return executeDispatch(
		ctx,
		h,
		HookSandboxSyncBefore,
		payload,
		dispatchConfig[SandboxSyncBeforePayload, SandboxSyncBeforePatch]{
			match:  matchSandboxSyncBefore,
			apply:  applySandboxSyncBeforePatch,
			denied: sandboxSyncBeforePatchDenied,
		},
	)
}

// DispatchSandboxSyncAfter runs the sandbox.sync.after hook dispatch.
func (h *Hooks) DispatchSandboxSyncAfter(
	ctx context.Context,
	payload SandboxSyncAfterPayload,
) (SandboxSyncAfterPayload, error) {
	return executeDispatch(
		ctx,
		h,
		HookSandboxSyncAfter,
		payload,
		dispatchConfig[SandboxSyncAfterPayload, SandboxSyncAfterPatch]{
			match: matchSandboxSyncAfter,
			apply: applyNoop[SandboxSyncAfterPayload, SandboxSyncAfterPatch],
		},
	)
}

// DispatchSandboxStop runs the sandbox.stop hook pipeline.
func (h *Hooks) DispatchSandboxStop(
	ctx context.Context,
	payload SandboxStopPayload,
) (SandboxStopPayload, error) {
	return executeDispatch(
		ctx,
		h,
		HookSandboxStop,
		payload,
		dispatchConfig[SandboxStopPayload, SandboxStopPatch]{
			match:  matchSandboxStop,
			apply:  applySandboxStopPatch,
			denied: sandboxStopPatchDenied,
		},
	)
}

// DispatchInputPreSubmit runs the input.pre_submit hook pipeline.
func (h *Hooks) DispatchInputPreSubmit(
	ctx context.Context,
	payload InputPreSubmitPayload,
) (InputPreSubmitPayload, error) {
	return executeDispatch(
		ctx,
		h,
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
		ctx,
		h,
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
func (h *Hooks) DispatchEventPreRecord(
	ctx context.Context,
	payload EventPreRecordPayload,
) (EventPreRecordPayload, error) {
	return executeDispatch(
		ctx,
		h,
		HookEventPreRecord,
		payload,
		dispatchConfig[EventPreRecordPayload, EventPreRecordPatch]{
			match: matchEventRecord,
			apply: applyNoop[EventPreRecordPayload, EventPreRecordPatch],
		},
	)
}

// DispatchEventPostRecord runs the event.post_record hook dispatch.
func (h *Hooks) DispatchEventPostRecord(
	ctx context.Context,
	payload EventPostRecordPayload,
) (EventPostRecordPayload, error) {
	return executeDispatch(
		ctx,
		h,
		HookEventPostRecord,
		payload,
		dispatchConfig[EventPostRecordPayload, EventPostRecordPatch]{
			match: matchEventRecord,
			apply: applyNoop[EventPostRecordPayload, EventPostRecordPatch],
		},
	)
}

// DispatchAutomationJobPreFire runs the automation.job.pre_fire hook pipeline.
func (h *Hooks) DispatchAutomationJobPreFire(
	ctx context.Context,
	payload AutomationJobPreFirePayload,
) (AutomationJobPreFirePayload, error) {
	return executeDispatch(
		ctx,
		h,
		HookAutomationJobPreFire,
		payload,
		dispatchConfig[AutomationJobPreFirePayload, AutomationFirePatch]{
			match:  matchAutomationJobPreFire,
			apply:  applyAutomationJobPreFirePatch,
			denied: automationFirePatchDenied,
			denyErr: func(AutomationJobPreFirePayload) error {
				return fmt.Errorf("%w: %s", ErrAutomationFireCancelled, HookAutomationJobPreFire)
			},
		},
	)
}

// DispatchAutomationJobPostFire runs the automation.job.post_fire hook dispatch.
func (h *Hooks) DispatchAutomationJobPostFire(
	ctx context.Context,
	payload AutomationJobPostFirePayload,
) (AutomationJobPostFirePayload, error) {
	return executeDispatch(
		ctx,
		h,
		HookAutomationJobPostFire,
		payload,
		dispatchConfig[AutomationJobPostFirePayload, AutomationObservationPatch]{
			match: matchAutomationJobPostFire,
			apply: applyNoop[AutomationJobPostFirePayload, AutomationObservationPatch],
		},
	)
}

// DispatchAutomationTriggerPreFire runs the automation.trigger.pre_fire hook pipeline.
func (h *Hooks) DispatchAutomationTriggerPreFire(
	ctx context.Context,
	payload AutomationTriggerPreFirePayload,
) (AutomationTriggerPreFirePayload, error) {
	return executeDispatch(
		ctx,
		h,
		HookAutomationTriggerPreFire,
		payload,
		dispatchConfig[AutomationTriggerPreFirePayload, AutomationFirePatch]{
			match:  matchAutomationTriggerPreFire,
			apply:  applyAutomationTriggerPreFirePatch,
			denied: automationFirePatchDenied,
			denyErr: func(AutomationTriggerPreFirePayload) error {
				return fmt.Errorf("%w: %s", ErrAutomationFireCancelled, HookAutomationTriggerPreFire)
			},
		},
	)
}

// DispatchAutomationTriggerPostFire runs the automation.trigger.post_fire hook dispatch.
func (h *Hooks) DispatchAutomationTriggerPostFire(
	ctx context.Context,
	payload AutomationTriggerPostFirePayload,
) (AutomationTriggerPostFirePayload, error) {
	return executeDispatch(
		ctx,
		h,
		HookAutomationTriggerPostFire,
		payload,
		dispatchConfig[AutomationTriggerPostFirePayload, AutomationObservationPatch]{
			match: matchAutomationTriggerPostFire,
			apply: applyNoop[AutomationTriggerPostFirePayload, AutomationObservationPatch],
		},
	)
}

// DispatchAutomationRunCompleted runs the automation.run.completed hook dispatch.
func (h *Hooks) DispatchAutomationRunCompleted(
	ctx context.Context,
	payload AutomationRunCompletedPayload,
) (AutomationRunCompletedPayload, error) {
	return executeDispatch(
		ctx,
		h,
		HookAutomationRunCompleted,
		payload,
		dispatchConfig[AutomationRunCompletedPayload, AutomationObservationPatch]{
			match: matchAutomationRunCompleted,
			apply: applyNoop[AutomationRunCompletedPayload, AutomationObservationPatch],
		},
	)
}

// DispatchAutomationRunFailed runs the automation.run.failed hook dispatch.
func (h *Hooks) DispatchAutomationRunFailed(
	ctx context.Context,
	payload AutomationRunFailedPayload,
) (AutomationRunFailedPayload, error) {
	return executeDispatch(
		ctx,
		h,
		HookAutomationRunFailed,
		payload,
		dispatchConfig[AutomationRunFailedPayload, AutomationObservationPatch]{
			match: matchAutomationRunFailed,
			apply: applyNoop[AutomationRunFailedPayload, AutomationObservationPatch],
		},
	)
}

// DispatchAgentPreStart runs the agent.pre_start hook pipeline.
func (h *Hooks) DispatchAgentPreStart(ctx context.Context, payload AgentPreStartPayload) (AgentPreStartPayload, error) {
	return executeDispatch(
		ctx,
		h,
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
		ctx,
		h,
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
		ctx,
		h,
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
		ctx,
		h,
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
		ctx,
		h,
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
		ctx,
		h,
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
		ctx,
		h,
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
		ctx,
		h,
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
		ctx,
		h,
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
		ctx,
		h,
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
		ctx,
		h,
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
		ctx,
		h,
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
func (h *Hooks) DispatchPermissionRequest(
	ctx context.Context,
	payload PermissionRequestPayload,
) (PermissionRequestPayload, error) {
	return executeDispatch(
		ctx,
		h,
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
func (h *Hooks) DispatchPermissionResolved(
	ctx context.Context,
	payload PermissionResolvedPayload,
) (PermissionResolvedPayload, error) {
	return executeDispatch(
		ctx,
		h,
		HookPermissionResolved,
		payload,
		dispatchConfig[PermissionResolvedPayload, PermissionResolvedPatch]{
			match: matchPermissionResolution,
			apply: applyNoop[PermissionResolvedPayload, PermissionResolvedPatch],
		},
	)
}

// DispatchPermissionDenied runs the permission.denied hook dispatch.
func (h *Hooks) DispatchPermissionDenied(
	ctx context.Context,
	payload PermissionDeniedPayload,
) (PermissionDeniedPayload, error) {
	return executeDispatch(
		ctx,
		h,
		HookPermissionDenied,
		payload,
		dispatchConfig[PermissionDeniedPayload, PermissionDeniedPatch]{
			match: matchPermissionResolution,
			apply: applyNoop[PermissionDeniedPayload, PermissionDeniedPatch],
		},
	)
}

// DispatchContextPreCompact runs the context.pre_compact hook pipeline.
func (h *Hooks) DispatchContextPreCompact(
	ctx context.Context,
	payload ContextPreCompactPayload,
) (ContextPreCompactPayload, error) {
	return executeDispatch(
		ctx,
		h,
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
func (h *Hooks) DispatchContextPostCompact(
	ctx context.Context,
	payload ContextPostCompactPayload,
) (ContextPostCompactPayload, error) {
	return executeDispatch(
		ctx,
		h,
		HookContextPostCompact,
		payload,
		dispatchConfig[ContextPostCompactPayload, ContextPostCompactPatch]{
			match:  matchContextCompact,
			apply:  applyContextCompactionPatch,
			denied: contextCompactionPatchDenied,
		},
	)
}

// DispatchCoordinatorPreSpawn runs the coordinator.pre_spawn hook pipeline.
func (h *Hooks) DispatchCoordinatorPreSpawn(
	ctx context.Context,
	payload CoordinatorPreSpawnPayload,
) (CoordinatorPreSpawnPayload, error) {
	return executeDispatch(
		ctx,
		h,
		HookCoordinatorPreSpawn,
		payload,
		dispatchConfig[CoordinatorPreSpawnPayload, CoordinatorSpawnPatch]{
			match:  matchCoordinatorPreSpawn,
			apply:  applyCoordinatorSpawnPatch,
			denied: coordinatorSpawnPatchDenied,
			denyErr: func(CoordinatorPreSpawnPayload) error {
				return fmt.Errorf("hooks: event %q denied", HookCoordinatorPreSpawn)
			},
		},
	)
}

// DispatchCoordinatorSpawned runs the coordinator.spawned hook dispatch.
func (h *Hooks) DispatchCoordinatorSpawned(
	ctx context.Context,
	payload CoordinatorSpawnedPayload,
) (CoordinatorSpawnedPayload, error) {
	return executeDispatch(
		ctx,
		h,
		HookCoordinatorSpawned,
		payload,
		dispatchConfig[CoordinatorSpawnedPayload, CoordinatorObservationPatch]{
			match: matchCoordinatorLifecycle,
			apply: applyNoop[CoordinatorSpawnedPayload, CoordinatorObservationPatch],
		},
	)
}

// DispatchCoordinatorDecision runs the coordinator.decision hook dispatch.
func (h *Hooks) DispatchCoordinatorDecision(
	ctx context.Context,
	payload CoordinatorDecisionPayload,
) (CoordinatorDecisionPayload, error) {
	return executeDispatch(
		ctx,
		h,
		HookCoordinatorDecision,
		payload,
		dispatchConfig[CoordinatorDecisionPayload, CoordinatorObservationPatch]{
			match: matchCoordinatorLifecycle,
			apply: applyNoop[CoordinatorDecisionPayload, CoordinatorObservationPatch],
		},
	)
}

// DispatchCoordinatorStopped runs the coordinator.stopped hook dispatch.
func (h *Hooks) DispatchCoordinatorStopped(
	ctx context.Context,
	payload CoordinatorStoppedPayload,
) (CoordinatorStoppedPayload, error) {
	return executeDispatch(
		ctx,
		h,
		HookCoordinatorStopped,
		payload,
		dispatchConfig[CoordinatorStoppedPayload, CoordinatorObservationPatch]{
			match: matchCoordinatorLifecycle,
			apply: applyNoop[CoordinatorStoppedPayload, CoordinatorObservationPatch],
		},
	)
}

// DispatchCoordinatorFailed runs the coordinator.failed hook dispatch.
func (h *Hooks) DispatchCoordinatorFailed(
	ctx context.Context,
	payload CoordinatorFailedPayload,
) (CoordinatorFailedPayload, error) {
	return executeDispatch(
		ctx,
		h,
		HookCoordinatorFailed,
		payload,
		dispatchConfig[CoordinatorFailedPayload, CoordinatorObservationPatch]{
			match: matchCoordinatorLifecycle,
			apply: applyNoop[CoordinatorFailedPayload, CoordinatorObservationPatch],
		},
	)
}

// DispatchTaskRunEnqueued runs the task.run.enqueued hook dispatch.
func (h *Hooks) DispatchTaskRunEnqueued(
	ctx context.Context,
	payload TaskRunEnqueuedPayload,
) (TaskRunEnqueuedPayload, error) {
	return executeDispatch(
		ctx,
		h,
		HookTaskRunEnqueued,
		payload,
		dispatchConfig[TaskRunEnqueuedPayload, TaskRunObservationPatch]{
			match: matchTaskRunEnqueued,
			apply: applyNoop[TaskRunEnqueuedPayload, TaskRunObservationPatch],
		},
	)
}

// DispatchTaskRunPreClaim runs the task.run.pre_claim hook pipeline.
func (h *Hooks) DispatchTaskRunPreClaim(
	ctx context.Context,
	payload TaskRunPreClaimPayload,
) (TaskRunPreClaimPayload, error) {
	return executeDispatch(
		ctx,
		h,
		HookTaskRunPreClaim,
		payload,
		dispatchConfig[TaskRunPreClaimPayload, TaskRunPreClaimPatch]{
			match:  matchTaskRunPreClaim,
			apply:  applyTaskRunPreClaimPatch,
			denied: taskRunPreClaimPatchDenied,
			denyErr: func(TaskRunPreClaimPayload) error {
				return fmt.Errorf("hooks: event %q denied", HookTaskRunPreClaim)
			},
			guard: guardTaskRunPreClaimPatch,
		},
	)
}

// DispatchTaskRunPostClaim runs the task.run.post_claim hook dispatch.
func (h *Hooks) DispatchTaskRunPostClaim(
	ctx context.Context,
	payload TaskRunPostClaimPayload,
) (TaskRunPostClaimPayload, error) {
	return executeDispatch(
		ctx,
		h,
		HookTaskRunPostClaim,
		payload,
		dispatchConfig[TaskRunPostClaimPayload, TaskRunObservationPatch]{
			match: matchTaskRunPostClaim,
			apply: applyNoop[TaskRunPostClaimPayload, TaskRunObservationPatch],
		},
	)
}

// DispatchTaskRunLeaseExtended runs the task.run.lease_extended hook dispatch.
func (h *Hooks) DispatchTaskRunLeaseExtended(
	ctx context.Context,
	payload TaskRunLeaseExtendedPayload,
) (TaskRunLeaseExtendedPayload, error) {
	return executeDispatch(
		ctx,
		h,
		HookTaskRunLeaseExtended,
		payload,
		dispatchConfig[TaskRunLeaseExtendedPayload, TaskRunObservationPatch]{
			match: matchTaskRunLease,
			apply: applyNoop[TaskRunLeaseExtendedPayload, TaskRunObservationPatch],
		},
	)
}

// DispatchTaskRunLeaseExpired runs the task.run.lease_expired hook dispatch.
func (h *Hooks) DispatchTaskRunLeaseExpired(
	ctx context.Context,
	payload TaskRunLeaseExpiredPayload,
) (TaskRunLeaseExpiredPayload, error) {
	return executeDispatch(
		ctx,
		h,
		HookTaskRunLeaseExpired,
		payload,
		dispatchConfig[TaskRunLeaseExpiredPayload, TaskRunObservationPatch]{
			match: matchTaskRunLease,
			apply: applyNoop[TaskRunLeaseExpiredPayload, TaskRunObservationPatch],
		},
	)
}

// DispatchTaskRunLeaseRecovered runs the task.run.lease_recovered hook dispatch.
func (h *Hooks) DispatchTaskRunLeaseRecovered(
	ctx context.Context,
	payload TaskRunLeaseRecoveredPayload,
) (TaskRunLeaseRecoveredPayload, error) {
	return executeDispatch(
		ctx,
		h,
		HookTaskRunLeaseRecovered,
		payload,
		dispatchConfig[TaskRunLeaseRecoveredPayload, TaskRunObservationPatch]{
			match: matchTaskRunLease,
			apply: applyNoop[TaskRunLeaseRecoveredPayload, TaskRunObservationPatch],
		},
	)
}

// DispatchTaskRunReleased runs the task.run.released hook dispatch.
func (h *Hooks) DispatchTaskRunReleased(
	ctx context.Context,
	payload TaskRunReleasedPayload,
) (TaskRunReleasedPayload, error) {
	return executeDispatch(
		ctx,
		h,
		HookTaskRunReleased,
		payload,
		dispatchConfig[TaskRunReleasedPayload, TaskRunObservationPatch]{
			match: matchTaskRunLease,
			apply: applyNoop[TaskRunReleasedPayload, TaskRunObservationPatch],
		},
	)
}

// DispatchTaskRunCompleted runs the task.run.completed hook dispatch.
func (h *Hooks) DispatchTaskRunCompleted(
	ctx context.Context,
	payload TaskRunCompletedPayload,
) (TaskRunCompletedPayload, error) {
	return executeDispatch(
		ctx,
		h,
		HookTaskRunCompleted,
		payload,
		dispatchConfig[TaskRunCompletedPayload, TaskRunObservationPatch]{
			match: matchTaskRunLease,
			apply: applyNoop[TaskRunCompletedPayload, TaskRunObservationPatch],
		},
	)
}

// DispatchTaskRunFailed runs the task.run.failed hook dispatch.
func (h *Hooks) DispatchTaskRunFailed(
	ctx context.Context,
	payload TaskRunFailedPayload,
) (TaskRunFailedPayload, error) {
	return executeDispatch(
		ctx,
		h,
		HookTaskRunFailed,
		payload,
		dispatchConfig[TaskRunFailedPayload, TaskRunObservationPatch]{
			match: matchTaskRunLease,
			apply: applyNoop[TaskRunFailedPayload, TaskRunObservationPatch],
		},
	)
}

// DispatchSpawnPreCreate runs the spawn.pre_create hook pipeline.
func (h *Hooks) DispatchSpawnPreCreate(
	ctx context.Context,
	payload SpawnPreCreatePayload,
) (SpawnPreCreatePayload, error) {
	return executeDispatch(
		ctx,
		h,
		HookSpawnPreCreate,
		payload,
		dispatchConfig[SpawnPreCreatePayload, SpawnCreatePatch]{
			match:  matchSpawnPreCreate,
			apply:  applySpawnCreatePatch,
			denied: spawnCreatePatchDenied,
			denyErr: func(SpawnPreCreatePayload) error {
				return fmt.Errorf("hooks: event %q denied", HookSpawnPreCreate)
			},
			guard: guardSpawnCreatePatch,
		},
	)
}

// DispatchSpawnCreated runs the spawn.created hook dispatch.
func (h *Hooks) DispatchSpawnCreated(
	ctx context.Context,
	payload SpawnCreatedPayload,
) (SpawnCreatedPayload, error) {
	return executeDispatch(
		ctx,
		h,
		HookSpawnCreated,
		payload,
		dispatchConfig[SpawnCreatedPayload, SpawnObservationPatch]{
			match: matchSpawnLifecycle,
			apply: applyNoop[SpawnCreatedPayload, SpawnObservationPatch],
		},
	)
}

// DispatchSpawnParentStopped runs the spawn.parent_stopped hook dispatch.
func (h *Hooks) DispatchSpawnParentStopped(
	ctx context.Context,
	payload SpawnParentStoppedPayload,
) (SpawnParentStoppedPayload, error) {
	return executeDispatch(
		ctx,
		h,
		HookSpawnParentStopped,
		payload,
		dispatchConfig[SpawnParentStoppedPayload, SpawnObservationPatch]{
			match: matchSpawnLifecycle,
			apply: applyNoop[SpawnParentStoppedPayload, SpawnObservationPatch],
		},
	)
}

// DispatchSpawnTTLExpired runs the spawn.ttl_expired hook dispatch.
func (h *Hooks) DispatchSpawnTTLExpired(
	ctx context.Context,
	payload SpawnTTLExpiredPayload,
) (SpawnTTLExpiredPayload, error) {
	return executeDispatch(
		ctx,
		h,
		HookSpawnTTLExpired,
		payload,
		dispatchConfig[SpawnTTLExpiredPayload, SpawnObservationPatch]{
			match: matchSpawnLifecycle,
			apply: applyNoop[SpawnTTLExpiredPayload, SpawnObservationPatch],
		},
	)
}

// DispatchSpawnReaped runs the spawn.reaped hook dispatch.
func (h *Hooks) DispatchSpawnReaped(
	ctx context.Context,
	payload SpawnReapedPayload,
) (SpawnReapedPayload, error) {
	return executeDispatch(
		ctx,
		h,
		HookSpawnReaped,
		payload,
		dispatchConfig[SpawnReapedPayload, SpawnObservationPatch]{
			match: matchSpawnLifecycle,
			apply: applyNoop[SpawnReapedPayload, SpawnObservationPatch],
		},
	)
}

func executeDispatch[P any, R any](
	ctx context.Context,
	h *Hooks,
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

	syncHooks, asyncHooks, err := matchingDispatchHooks(h, event, payload, cfg.match)
	if err != nil {
		return payload, err
	}
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
		submitAsyncHooks(ctx, h, result, asyncHooks, pipe)
	}

	reportDispatchResult(h, event, dispatchDepth, dispatchStarted, report, dispatchErr, len(syncHooks), len(asyncHooks))

	return result, dispatchErr
}

func matchingDispatchHooks[P any](
	h *Hooks,
	event HookEvent,
	payload P,
	match matcherFunc[P],
) ([]*ResolvedHook, []*ResolvedHook, error) {
	snapshot, err := h.hookSnapshot(event)
	if err != nil {
		return nil, nil, err
	}
	syncHooks, asyncHooks := selectMatchingHooks(snapshot, payload, match)
	return syncHooks, asyncHooks, nil
}

func reportDispatchResult(
	h *Hooks,
	event HookEvent,
	dispatchDepth int,
	dispatchStarted time.Time,
	report dispatchReport,
	dispatchErr error,
	syncHookCount int,
	asyncHookCount int,
) {
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
			"sync_hooks", syncHookCount,
			"async_hooks", asyncHookCount,
		)
	}
}

func applyNoop[P any, R any](payload P, _ R) P {
	return payload
}

func automationFirePatchDenied(patch AutomationFirePatch) bool {
	return patch.Cancel
}

func applyAutomationJobPreFirePatch(
	payload AutomationJobPreFirePayload,
	patch AutomationFirePatch,
) AutomationJobPreFirePayload {
	if patch.Prompt != nil {
		payload.Prompt = *patch.Prompt
	}
	return payload
}

func applyAutomationTriggerPreFirePatch(
	payload AutomationTriggerPreFirePayload,
	patch AutomationFirePatch,
) AutomationTriggerPreFirePayload {
	if patch.Prompt != nil {
		payload.Prompt = *patch.Prompt
	}
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

func applySandboxPreparePatch(
	payload SandboxPreparePayload,
	patch SandboxPreparePatch,
) SandboxPreparePayload {
	if patch.Deny {
		payload.Denied = true
		payload.DenyReason = patch.DenyReason
	}
	if patch.EnvOverrides != nil {
		payload.EnvOverrides = cloneStringMap(patch.EnvOverrides)
	}
	return payload
}

func applySandboxSyncBeforePatch(
	payload SandboxSyncBeforePayload,
	patch SandboxSyncBeforePatch,
) SandboxSyncBeforePayload {
	if patch.Deny {
		payload.Denied = true
		payload.DenyReason = patch.DenyReason
	}
	if patch.ExcludePatterns != nil {
		payload.ExcludePatterns = append([]string(nil), patch.ExcludePatterns...)
	}
	return payload
}

func applySandboxStopPatch(payload SandboxStopPayload, patch SandboxStopPatch) SandboxStopPayload {
	if patch.Deny {
		payload.Denied = true
		payload.DenyReason = patch.DenyReason
	}
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
	if patch.ToolID != nil {
		payload.ToolID = *patch.ToolID
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

func mergePermissionRequestPatch(
	payload PermissionRequestPayload,
	patch PermissionRequestPatch,
) PermissionRequestPayload {
	if patch.Decision != nil {
		payload.Decision = *patch.Decision
	}
	if patch.Deny {
		payload.Decision = permissionDecisionDeny
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

func applyCoordinatorSpawnPatch(
	payload CoordinatorPreSpawnPayload,
	patch CoordinatorSpawnPatch,
) CoordinatorPreSpawnPayload {
	if patch.Deny {
		payload.Denied = true
		payload.DenyReason = patch.DenyReason
	}
	if patch.AgentName != nil {
		payload.AgentName = strings.TrimSpace(*patch.AgentName)
	}
	if patch.Provider != nil {
		payload.Provider = strings.TrimSpace(*patch.Provider)
	}
	if patch.Model != nil {
		payload.Model = strings.TrimSpace(*patch.Model)
	}
	return payload
}

func applyTaskRunPreClaimPatch(
	payload TaskRunPreClaimPayload,
	patch TaskRunPreClaimPatch,
) TaskRunPreClaimPayload {
	if patch.Deny {
		payload.Denied = true
		payload.DenyReason = patch.DenyReason
	}
	if patch.AddRequiredCapabilities != nil {
		payload.Criteria.RequiredCapabilities = unionStringSet(
			payload.Criteria.RequiredCapabilities,
			patch.AddRequiredCapabilities,
		)
	}
	if patch.PriorityMin != nil && *patch.PriorityMin > payload.Criteria.PriorityMin {
		payload.Criteria.PriorityMin = *patch.PriorityMin
	}
	return payload
}

func applySpawnCreatePatch(payload SpawnPreCreatePayload, patch SpawnCreatePatch) SpawnPreCreatePayload {
	if patch.Deny {
		payload.Denied = true
		payload.DenyReason = patch.DenyReason
	}
	if patch.AgentName != nil {
		payload.AgentName = strings.TrimSpace(*patch.AgentName)
	}
	if patch.SpawnRole != nil {
		payload.SpawnRole = strings.TrimSpace(*patch.SpawnRole)
	}
	if patch.TTLSeconds != nil {
		payload.TTLSeconds = *patch.TTLSeconds
	}
	if patch.ChildPermissions != nil {
		payload.ChildPermissions = normalizePermissionSet(patch.ChildPermissions)
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

func sandboxPreparePatchDenied(patch SandboxPreparePatch) bool {
	return patch.Deny
}

func sandboxSyncBeforePatchDenied(patch SandboxSyncBeforePatch) bool {
	return patch.Deny
}

func sandboxStopPatchDenied(patch SandboxStopPatch) bool {
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

func coordinatorSpawnPatchDenied(patch CoordinatorSpawnPatch) bool {
	return patch.Deny
}

func taskRunPreClaimPatchDenied(patch TaskRunPreClaimPatch) bool {
	return patch.Deny
}

func spawnCreatePatchDenied(patch SpawnCreatePatch) bool {
	return patch.Deny
}

func guardTaskRunPreClaimPatch(
	_ context.Context,
	hook RegisteredHook,
	payload TaskRunPreClaimPayload,
	patch TaskRunPreClaimPatch,
) error {
	if patch.PriorityMin != nil && *patch.PriorityMin < payload.Criteria.PriorityMin {
		return fmt.Errorf(
			"%w: hook %q cannot lower task-run claim priority_min from %d to %d",
			ErrHookPatchRejected,
			hook.Name,
			payload.Criteria.PriorityMin,
			*patch.PriorityMin,
		)
	}
	for _, capability := range patch.AddRequiredCapabilities {
		if strings.TrimSpace(capability) == "" {
			return fmt.Errorf(
				"%w: hook %q cannot add blank task-run claim capability",
				ErrHookPatchRejected,
				hook.Name,
			)
		}
	}
	return nil
}

func guardSpawnCreatePatch(
	_ context.Context,
	hook RegisteredHook,
	payload SpawnPreCreatePayload,
	patch SpawnCreatePatch,
) error {
	if patch.TTLSeconds != nil && *patch.TTLSeconds <= 0 {
		return fmt.Errorf(
			"%w: hook %q cannot set non-positive spawn ttl_seconds",
			ErrHookPatchRejected,
			hook.Name,
		)
	}

	child := payload.ChildPermissions
	if patch.ChildPermissions != nil {
		child = patch.ChildPermissions
	}
	if err := validatePermissionSubset(payload.ParentPermissions, child); err != nil {
		return fmt.Errorf("%w: hook %q spawn permission patch rejected: %w", ErrHookPatchRejected, hook.Name, err)
	}
	return nil
}

func validatePermissionSubset(parent *PermissionSet, child *PermissionSet) error {
	if err := validatePermissionAtoms("tools", permissionTools(parent), permissionTools(child)); err != nil {
		return err
	}
	if err := validatePermissionAtoms("skills", permissionSkills(parent), permissionSkills(child)); err != nil {
		return err
	}
	if err := validatePermissionAtoms(
		"mcp_servers",
		permissionMCPServers(parent),
		permissionMCPServers(child),
	); err != nil {
		return err
	}
	if err := validatePermissionAtoms(
		"workspace_paths",
		permissionWorkspacePaths(parent),
		permissionWorkspacePaths(child),
	); err != nil {
		return err
	}
	if err := validatePermissionAtoms(
		"network_channels",
		permissionNetworkChannels(parent),
		permissionNetworkChannels(child),
	); err != nil {
		return err
	}
	return validatePermissionAtoms(
		"sandbox_profiles",
		permissionSandboxProfiles(parent),
		permissionSandboxProfiles(child),
	)
}

func permissionTools(src *PermissionSet) []string {
	if src == nil {
		return nil
	}
	return src.Tools
}

func permissionSkills(src *PermissionSet) []string {
	if src == nil {
		return nil
	}
	return src.Skills
}

func permissionMCPServers(src *PermissionSet) []string {
	if src == nil {
		return nil
	}
	return src.MCPServers
}

func permissionWorkspacePaths(src *PermissionSet) []string {
	if src == nil {
		return nil
	}
	return src.WorkspacePaths
}

func permissionNetworkChannels(src *PermissionSet) []string {
	if src == nil {
		return nil
	}
	return src.NetworkChannels
}

func permissionSandboxProfiles(src *PermissionSet) []string {
	if src == nil {
		return nil
	}
	return src.SandboxProfiles
}

func validatePermissionAtoms(category string, parent []string, child []string) error {
	allowed := make(map[string]struct{}, len(parent))
	for _, atom := range parent {
		trimmed := strings.TrimSpace(atom)
		if trimmed == "" {
			return fmt.Errorf("parent %s includes a blank permission atom", category)
		}
		allowed[trimmed] = struct{}{}
	}
	for _, atom := range child {
		trimmed := strings.TrimSpace(atom)
		if trimmed == "" {
			return fmt.Errorf("child %s includes a blank permission atom", category)
		}
		if _, ok := allowed[trimmed]; !ok {
			return fmt.Errorf("child %s permission atom %q widens parent permissions", category, trimmed)
		}
	}
	return nil
}

func normalizePermissionSet(src *PermissionSet) *PermissionSet {
	if src == nil {
		return nil
	}
	return &PermissionSet{
		Tools:           uniqueTrimmedStrings(src.Tools),
		Skills:          uniqueTrimmedStrings(src.Skills),
		MCPServers:      uniqueTrimmedStrings(src.MCPServers),
		WorkspacePaths:  uniqueTrimmedStrings(src.WorkspacePaths),
		NetworkChannels: uniqueTrimmedStrings(src.NetworkChannels),
		SandboxProfiles: uniqueTrimmedStrings(src.SandboxProfiles),
	}
}

func unionStringSet(base []string, additions []string) []string {
	return uniqueTrimmedStrings(append(append([]string(nil), base...), additions...))
}

func uniqueTrimmedStrings(values []string) []string {
	if values == nil {
		return nil
	}
	out := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		out = append(out, trimmed)
	}
	return out
}
