package daemon

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"slices"
	"strings"
	"sync"
	"time"

	aghconfig "github.com/pedronauck/agh/internal/config"
	hookspkg "github.com/pedronauck/agh/internal/hooks"
	"github.com/pedronauck/agh/internal/session"
	"github.com/pedronauck/agh/internal/skills"
	taskpkg "github.com/pedronauck/agh/internal/task"
	"github.com/pedronauck/agh/internal/toolruntime"
	workspacepkg "github.com/pedronauck/agh/internal/workspace"
)

type hookRuntime interface {
	Close()
	Version() int64
	DispatchSessionPreCreate(
		context.Context,
		hookspkg.SessionPreCreatePayload,
	) (hookspkg.SessionPreCreatePayload, error)
	DispatchSessionPostCreate(
		context.Context,
		hookspkg.SessionPostCreatePayload,
	) (hookspkg.SessionPostCreatePayload, error)
	DispatchSessionPreResume(
		context.Context,
		hookspkg.SessionPreResumePayload,
	) (hookspkg.SessionPreResumePayload, error)
	DispatchSessionPostResume(
		context.Context,
		hookspkg.SessionPostResumePayload,
	) (hookspkg.SessionPostResumePayload, error)
	DispatchSessionPreStop(context.Context, hookspkg.SessionPreStopPayload) (hookspkg.SessionPreStopPayload, error)
	DispatchSessionPostStop(context.Context, hookspkg.SessionPostStopPayload) (hookspkg.SessionPostStopPayload, error)
	DispatchSandboxPrepare(
		context.Context,
		hookspkg.SandboxPreparePayload,
	) (hookspkg.SandboxPreparePayload, error)
	DispatchSandboxReady(
		context.Context,
		hookspkg.SandboxReadyPayload,
	) (hookspkg.SandboxReadyPayload, error)
	DispatchSandboxSyncBefore(
		context.Context,
		hookspkg.SandboxSyncBeforePayload,
	) (hookspkg.SandboxSyncBeforePayload, error)
	DispatchSandboxSyncAfter(
		context.Context,
		hookspkg.SandboxSyncAfterPayload,
	) (hookspkg.SandboxSyncAfterPayload, error)
	DispatchSandboxStop(context.Context, hookspkg.SandboxStopPayload) (hookspkg.SandboxStopPayload, error)
	DispatchInputPreSubmit(context.Context, hookspkg.InputPreSubmitPayload) (hookspkg.InputPreSubmitPayload, error)
	DispatchPromptPostAssemble(context.Context, hookspkg.PromptPayload) (hookspkg.PromptPayload, error)
	DispatchEventPreRecord(context.Context, hookspkg.EventPreRecordPayload) (hookspkg.EventPreRecordPayload, error)
	DispatchEventPostRecord(context.Context, hookspkg.EventPostRecordPayload) (hookspkg.EventPostRecordPayload, error)
	DispatchAutomationJobPreFire(
		context.Context,
		hookspkg.AutomationJobPreFirePayload,
	) (hookspkg.AutomationJobPreFirePayload, error)
	DispatchAutomationJobPostFire(
		context.Context,
		hookspkg.AutomationJobPostFirePayload,
	) (hookspkg.AutomationJobPostFirePayload, error)
	DispatchAutomationTriggerPreFire(
		context.Context,
		hookspkg.AutomationTriggerPreFirePayload,
	) (hookspkg.AutomationTriggerPreFirePayload, error)
	DispatchAutomationTriggerPostFire(
		context.Context,
		hookspkg.AutomationTriggerPostFirePayload,
	) (hookspkg.AutomationTriggerPostFirePayload, error)
	DispatchAutomationRunCompleted(
		context.Context,
		hookspkg.AutomationRunCompletedPayload,
	) (hookspkg.AutomationRunCompletedPayload, error)
	DispatchAutomationRunFailed(
		context.Context,
		hookspkg.AutomationRunFailedPayload,
	) (hookspkg.AutomationRunFailedPayload, error)
	DispatchAgentPreStart(context.Context, hookspkg.AgentPreStartPayload) (hookspkg.AgentPreStartPayload, error)
	DispatchAgentSpawned(context.Context, hookspkg.AgentSpawnedPayload) (hookspkg.AgentSpawnedPayload, error)
	DispatchAgentCrashed(context.Context, hookspkg.AgentCrashedPayload) (hookspkg.AgentCrashedPayload, error)
	DispatchAgentStopped(context.Context, hookspkg.AgentStoppedPayload) (hookspkg.AgentStoppedPayload, error)
	DispatchTurnStart(context.Context, hookspkg.TurnStartPayload) (hookspkg.TurnStartPayload, error)
	DispatchTurnEnd(context.Context, hookspkg.TurnEndPayload) (hookspkg.TurnEndPayload, error)
	DispatchMessageStart(context.Context, hookspkg.MessageStartPayload) (hookspkg.MessageStartPayload, error)
	DispatchMessageDelta(context.Context, hookspkg.MessageDeltaPayload) (hookspkg.MessageDeltaPayload, error)
	DispatchMessageEnd(context.Context, hookspkg.MessageEndPayload) (hookspkg.MessageEndPayload, error)
	DispatchToolPreCall(context.Context, hookspkg.ToolPreCallPayload) (hookspkg.ToolPreCallPayload, error)
	DispatchToolPostCall(context.Context, hookspkg.ToolPostCallPayload) (hookspkg.ToolPostCallPayload, error)
	DispatchToolPostError(context.Context, hookspkg.ToolPostErrorPayload) (hookspkg.ToolPostErrorPayload, error)
	DispatchPermissionRequest(
		context.Context,
		hookspkg.PermissionRequestPayload,
	) (hookspkg.PermissionRequestPayload, error)
	DispatchPermissionResolved(
		context.Context,
		hookspkg.PermissionResolvedPayload,
	) (hookspkg.PermissionResolvedPayload, error)
	DispatchPermissionDenied(
		context.Context,
		hookspkg.PermissionDeniedPayload,
	) (hookspkg.PermissionDeniedPayload, error)
	DispatchContextPreCompact(
		context.Context,
		hookspkg.ContextPreCompactPayload,
	) (hookspkg.ContextPreCompactPayload, error)
	DispatchContextPostCompact(
		context.Context,
		hookspkg.ContextPostCompactPayload,
	) (hookspkg.ContextPostCompactPayload, error)
	DispatchCoordinatorPreSpawn(
		context.Context,
		hookspkg.CoordinatorPreSpawnPayload,
	) (hookspkg.CoordinatorPreSpawnPayload, error)
	DispatchCoordinatorSpawned(
		context.Context,
		hookspkg.CoordinatorSpawnedPayload,
	) (hookspkg.CoordinatorSpawnedPayload, error)
	DispatchCoordinatorDecision(
		context.Context,
		hookspkg.CoordinatorDecisionPayload,
	) (hookspkg.CoordinatorDecisionPayload, error)
	DispatchCoordinatorStopped(
		context.Context,
		hookspkg.CoordinatorStoppedPayload,
	) (hookspkg.CoordinatorStoppedPayload, error)
	DispatchCoordinatorFailed(
		context.Context,
		hookspkg.CoordinatorFailedPayload,
	) (hookspkg.CoordinatorFailedPayload, error)
	DispatchTaskRunEnqueued(
		context.Context,
		hookspkg.TaskRunEnqueuedPayload,
	) (hookspkg.TaskRunEnqueuedPayload, error)
	DispatchTaskRunPreClaim(
		context.Context,
		hookspkg.TaskRunPreClaimPayload,
	) (hookspkg.TaskRunPreClaimPayload, error)
	DispatchTaskRunPostClaim(
		context.Context,
		hookspkg.TaskRunPostClaimPayload,
	) (hookspkg.TaskRunPostClaimPayload, error)
	DispatchTaskRunLeaseExtended(
		context.Context,
		hookspkg.TaskRunLeaseExtendedPayload,
	) (hookspkg.TaskRunLeaseExtendedPayload, error)
	DispatchTaskRunLeaseExpired(
		context.Context,
		hookspkg.TaskRunLeaseExpiredPayload,
	) (hookspkg.TaskRunLeaseExpiredPayload, error)
	DispatchTaskRunLeaseRecovered(
		context.Context,
		hookspkg.TaskRunLeaseRecoveredPayload,
	) (hookspkg.TaskRunLeaseRecoveredPayload, error)
	DispatchTaskRunReleased(
		context.Context,
		hookspkg.TaskRunReleasedPayload,
	) (hookspkg.TaskRunReleasedPayload, error)
	DispatchTaskRunCompleted(
		context.Context,
		hookspkg.TaskRunCompletedPayload,
	) (hookspkg.TaskRunCompletedPayload, error)
	DispatchTaskRunFailed(
		context.Context,
		hookspkg.TaskRunFailedPayload,
	) (hookspkg.TaskRunFailedPayload, error)
	DispatchSpawnPreCreate(
		context.Context,
		hookspkg.SpawnPreCreatePayload,
	) (hookspkg.SpawnPreCreatePayload, error)
	DispatchSpawnCreated(
		context.Context,
		hookspkg.SpawnCreatedPayload,
	) (hookspkg.SpawnCreatedPayload, error)
	DispatchSpawnParentStopped(
		context.Context,
		hookspkg.SpawnParentStoppedPayload,
	) (hookspkg.SpawnParentStoppedPayload, error)
	DispatchSpawnTTLExpired(
		context.Context,
		hookspkg.SpawnTTLExpiredPayload,
	) (hookspkg.SpawnTTLExpiredPayload, error)
	DispatchSpawnReaped(
		context.Context,
		hookspkg.SpawnReapedPayload,
	) (hookspkg.SpawnReapedPayload, error)
	DispatchAgentSoulSnapshotResolved(
		context.Context,
		hookspkg.AgentSoulSnapshotResolvedPayload,
	) (hookspkg.AgentSoulSnapshotResolvedPayload, error)
	DispatchAgentSoulMutationAfter(
		context.Context,
		hookspkg.AgentSoulMutationAfterPayload,
	) (hookspkg.AgentSoulMutationAfterPayload, error)
	DispatchAgentHeartbeatPolicyResolved(
		context.Context,
		hookspkg.AgentHeartbeatPolicyResolvedPayload,
	) (hookspkg.AgentHeartbeatPolicyResolvedPayload, error)
	DispatchAgentHeartbeatWakeBefore(
		context.Context,
		hookspkg.AgentHeartbeatWakeBeforePayload,
	) (hookspkg.AgentHeartbeatWakeBeforePayload, error)
	DispatchAgentHeartbeatWakeAfter(
		context.Context,
		hookspkg.AgentHeartbeatWakeAfterPayload,
	) (hookspkg.AgentHeartbeatWakeAfterPayload, error)
	DispatchSessionHealthUpdateAfter(
		context.Context,
		hookspkg.SessionHealthUpdateAfterPayload,
	) (hookspkg.SessionHealthUpdateAfterPayload, error)
}

type sessionLifecycleObserver interface {
	OnSessionCreated(context.Context, *session.Session)
	OnSessionStopped(context.Context, *session.Session)
}

type taskRunEnqueuedObserver interface {
	OnTaskRunEnqueued(context.Context, hookspkg.TaskRunEnqueuedPayload)
}

type dreamCheckEnqueuer interface {
	EnqueueCheck(reason string, workspaceRef string)
}

type sessionLifecycleFanout struct {
	mu        sync.RWMutex
	observers []sessionLifecycleObserver
}

func newSessionLifecycleFanout(observers ...sessionLifecycleObserver) *sessionLifecycleFanout {
	fanout := &sessionLifecycleFanout{}
	for _, observer := range observers {
		fanout.Add(observer)
	}
	return fanout
}

func (f *sessionLifecycleFanout) Add(observer sessionLifecycleObserver) {
	if f == nil || observer == nil {
		return
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.observers = append(f.observers, observer)
}

func (f *sessionLifecycleFanout) OnSessionCreated(ctx context.Context, sess *session.Session) {
	for _, observer := range f.snapshot() {
		observer.OnSessionCreated(ctx, sess)
	}
}

func (f *sessionLifecycleFanout) OnSessionStopped(ctx context.Context, sess *session.Session) {
	for _, observer := range f.snapshot() {
		observer.OnSessionStopped(ctx, sess)
	}
}

func (f *sessionLifecycleFanout) snapshot() []sessionLifecycleObserver {
	if f == nil {
		return nil
	}
	f.mu.RLock()
	defer f.mu.RUnlock()
	return append([]sessionLifecycleObserver(nil), f.observers...)
}

type hookTelemetryFanout struct {
	mu    sync.RWMutex
	sinks []hookspkg.TelemetrySink
}

func newHookTelemetryFanout(sinks ...hookspkg.TelemetrySink) *hookTelemetryFanout {
	fanout := &hookTelemetryFanout{}
	for _, sink := range sinks {
		fanout.Add(sink)
	}
	return fanout
}

func (f *hookTelemetryFanout) Add(sink hookspkg.TelemetrySink) {
	if f == nil || sink == nil {
		return
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.sinks = append(f.sinks, sink)
}

func (f *hookTelemetryFanout) WriteHookRecord(
	ctx context.Context,
	sessionID string,
	record hookspkg.HookRunRecord,
) error {
	var errs []error
	for _, sink := range f.snapshot() {
		if err := sink.WriteHookRecord(ctx, sessionID, record); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func (f *hookTelemetryFanout) snapshot() []hookspkg.TelemetrySink {
	if f == nil {
		return nil
	}
	f.mu.RLock()
	defer f.mu.RUnlock()
	return append([]hookspkg.TelemetrySink(nil), f.sinks...)
}

type hooksNotifier struct {
	mu sync.RWMutex

	logger               *slog.Logger
	now                  func() time.Time
	hooks                hookRuntime
	agentEventNotify     session.Notifier
	taskRunEnqueuedHooks []taskRunEnqueuedObserver
}

var _ session.Notifier = (*hooksNotifier)(nil)
var _ session.LifecycleHooks = (*hooksNotifier)(nil)
var _ session.SandboxHooks = (*hooksNotifier)(nil)
var _ session.PromptHooks = (*hooksNotifier)(nil)
var _ session.EventHooks = (*hooksNotifier)(nil)
var _ session.AgentHooks = (*hooksNotifier)(nil)
var _ session.ConversationHooks = (*hooksNotifier)(nil)
var _ session.CompactionHooks = (*hooksNotifier)(nil)
var _ session.SpawnHooks = (*hooksNotifier)(nil)
var _ session.AuthoredContextHooks = (*hooksNotifier)(nil)
var _ taskpkg.RunHookDispatcher = (*hooksNotifier)(nil)
var _ session.AgentEventNotifier = (*hooksNotifier)(nil)
var _ session.SandboxLifecycleNotifier = (*hooksNotifier)(nil)

func newHooksNotifier(logger *slog.Logger, now func() time.Time) *hooksNotifier {
	if logger == nil {
		logger = slog.Default()
	}
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}

	return &hooksNotifier{
		logger: logger,
		now:    now,
	}
}

func (n *hooksNotifier) setRuntime(hooks hookRuntime, agentEventNotify session.Notifier) {
	n.mu.Lock()
	defer n.mu.Unlock()

	n.hooks = hooks
	n.agentEventNotify = agentEventNotify
}

func (n *hooksNotifier) AddTaskRunEnqueuedObserver(observer taskRunEnqueuedObserver) {
	if n == nil || observer == nil {
		return
	}
	n.mu.Lock()
	defer n.mu.Unlock()
	n.taskRunEnqueuedHooks = append(n.taskRunEnqueuedHooks, observer)
}

func (n *hooksNotifier) taskRunEnqueuedObservers() []taskRunEnqueuedObserver {
	if n == nil {
		return nil
	}
	n.mu.RLock()
	defer n.mu.RUnlock()
	return append([]taskRunEnqueuedObserver(nil), n.taskRunEnqueuedHooks...)
}

// OnSessionCreated forwards the full runtime session to the downstream
// observer after hook dispatch has already run. The native hook payload keeps
// lifecycle ordering, while this pass preserves catalog fields that are not
// exposed on public hook payloads.
func (n *hooksNotifier) OnSessionCreated(ctx context.Context, sess *session.Session) {
	if sess == nil {
		return
	}
	_, agentEventNotify := n.runtime()
	if agentEventNotify != nil {
		agentEventNotify.OnSessionCreated(ctx, sess)
	}
}

// OnSessionStopped forwards the full runtime session to the downstream
// observer after hook dispatch has already run.
func (n *hooksNotifier) OnSessionStopped(ctx context.Context, sess *session.Session) {
	if sess == nil {
		return
	}
	_, agentEventNotify := n.runtime()
	if agentEventNotify != nil {
		agentEventNotify.OnSessionStopped(ctx, sess)
	}
}

func (n *hooksNotifier) DispatchSessionPreCreate(
	ctx context.Context,
	payload hookspkg.SessionPreCreatePayload,
) (hookspkg.SessionPreCreatePayload, error) {
	return dispatchRuntime(
		ctx,
		n,
		hookspkg.HookSessionPreCreate,
		payload,
		hookRuntime.DispatchSessionPreCreate,
	)
}

func (n *hooksNotifier) DispatchSessionPostCreate(
	ctx context.Context,
	payload hookspkg.SessionPostCreatePayload,
) (hookspkg.SessionPostCreatePayload, error) {
	return dispatchRuntime(
		ctx,
		n,
		hookspkg.HookSessionPostCreate,
		payload,
		hookRuntime.DispatchSessionPostCreate,
	)
}

func (n *hooksNotifier) DispatchSessionPreResume(
	ctx context.Context,
	payload hookspkg.SessionPreResumePayload,
) (hookspkg.SessionPreResumePayload, error) {
	return dispatchRuntime(
		ctx,
		n,
		hookspkg.HookSessionPreResume,
		payload,
		hookRuntime.DispatchSessionPreResume,
	)
}

func (n *hooksNotifier) DispatchSessionPostResume(
	ctx context.Context,
	payload hookspkg.SessionPostResumePayload,
) (hookspkg.SessionPostResumePayload, error) {
	return dispatchRuntime(
		ctx,
		n,
		hookspkg.HookSessionPostResume,
		payload,
		hookRuntime.DispatchSessionPostResume,
	)
}

func (n *hooksNotifier) DispatchSessionPreStop(
	ctx context.Context,
	payload hookspkg.SessionPreStopPayload,
) (hookspkg.SessionPreStopPayload, error) {
	return dispatchRuntime(
		ctx,
		n,
		hookspkg.HookSessionPreStop,
		payload,
		hookRuntime.DispatchSessionPreStop,
	)
}

func (n *hooksNotifier) DispatchSessionPostStop(
	ctx context.Context,
	payload hookspkg.SessionPostStopPayload,
) (hookspkg.SessionPostStopPayload, error) {
	return dispatchRuntime(
		ctx,
		n,
		hookspkg.HookSessionPostStop,
		payload,
		hookRuntime.DispatchSessionPostStop,
	)
}

func (n *hooksNotifier) DispatchSandboxPrepare(
	ctx context.Context,
	payload hookspkg.SandboxPreparePayload,
) (hookspkg.SandboxPreparePayload, error) {
	return dispatchRuntime(
		ctx,
		n,
		hookspkg.HookSandboxPrepare,
		payload,
		hookRuntime.DispatchSandboxPrepare,
	)
}

func (n *hooksNotifier) DispatchSandboxReady(
	ctx context.Context,
	payload hookspkg.SandboxReadyPayload,
) (hookspkg.SandboxReadyPayload, error) {
	return dispatchRuntime(
		ctx,
		n,
		hookspkg.HookSandboxReady,
		payload,
		hookRuntime.DispatchSandboxReady,
	)
}

func (n *hooksNotifier) DispatchSandboxSyncBefore(
	ctx context.Context,
	payload hookspkg.SandboxSyncBeforePayload,
) (hookspkg.SandboxSyncBeforePayload, error) {
	return dispatchRuntime(
		ctx,
		n,
		hookspkg.HookSandboxSyncBefore,
		payload,
		hookRuntime.DispatchSandboxSyncBefore,
	)
}

func (n *hooksNotifier) DispatchSandboxSyncAfter(
	ctx context.Context,
	payload hookspkg.SandboxSyncAfterPayload,
) (hookspkg.SandboxSyncAfterPayload, error) {
	return dispatchRuntime(
		ctx,
		n,
		hookspkg.HookSandboxSyncAfter,
		payload,
		hookRuntime.DispatchSandboxSyncAfter,
	)
}

func (n *hooksNotifier) DispatchSandboxStop(
	ctx context.Context,
	payload hookspkg.SandboxStopPayload,
) (hookspkg.SandboxStopPayload, error) {
	return dispatchRuntime(
		ctx,
		n,
		hookspkg.HookSandboxStop,
		payload,
		hookRuntime.DispatchSandboxStop,
	)
}

func (n *hooksNotifier) DispatchInputPreSubmit(
	ctx context.Context,
	payload hookspkg.InputPreSubmitPayload,
) (hookspkg.InputPreSubmitPayload, error) {
	return dispatchRuntime(
		ctx,
		n,
		hookspkg.HookInputPreSubmit,
		payload,
		hookRuntime.DispatchInputPreSubmit,
	)
}

func (n *hooksNotifier) DispatchPromptPostAssemble(
	ctx context.Context,
	payload hookspkg.PromptPayload,
) (hookspkg.PromptPayload, error) {
	return dispatchRuntime(
		ctx,
		n,
		hookspkg.HookPromptPostAssemble,
		payload,
		hookRuntime.DispatchPromptPostAssemble,
	)
}

func (n *hooksNotifier) DispatchEventPreRecord(
	ctx context.Context,
	payload hookspkg.EventPreRecordPayload,
) (hookspkg.EventPreRecordPayload, error) {
	return dispatchRuntime(
		ctx,
		n,
		hookspkg.HookEventPreRecord,
		payload,
		hookRuntime.DispatchEventPreRecord,
	)
}

func (n *hooksNotifier) DispatchEventPostRecord(
	ctx context.Context,
	payload hookspkg.EventPostRecordPayload,
) (hookspkg.EventPostRecordPayload, error) {
	return dispatchRuntime(
		ctx,
		n,
		hookspkg.HookEventPostRecord,
		payload,
		hookRuntime.DispatchEventPostRecord,
	)
}

func (n *hooksNotifier) DispatchAgentPreStart(
	ctx context.Context,
	payload hookspkg.AgentPreStartPayload,
) (hookspkg.AgentPreStartPayload, error) {
	return dispatchRuntime(
		ctx,
		n,
		hookspkg.HookAgentPreStart,
		payload,
		hookRuntime.DispatchAgentPreStart,
	)
}

func (n *hooksNotifier) DispatchAgentSpawned(
	ctx context.Context,
	payload hookspkg.AgentSpawnedPayload,
) (hookspkg.AgentSpawnedPayload, error) {
	return dispatchRuntime(
		ctx,
		n,
		hookspkg.HookAgentSpawned,
		payload,
		hookRuntime.DispatchAgentSpawned,
	)
}

func (n *hooksNotifier) DispatchAgentCrashed(
	ctx context.Context,
	payload hookspkg.AgentCrashedPayload,
) (hookspkg.AgentCrashedPayload, error) {
	return dispatchRuntime(
		ctx,
		n,
		hookspkg.HookAgentCrashed,
		payload,
		hookRuntime.DispatchAgentCrashed,
	)
}

func (n *hooksNotifier) DispatchAgentStopped(
	ctx context.Context,
	payload hookspkg.AgentStoppedPayload,
) (hookspkg.AgentStoppedPayload, error) {
	return dispatchRuntime(
		ctx,
		n,
		hookspkg.HookAgentStopped,
		payload,
		hookRuntime.DispatchAgentStopped,
	)
}

func (n *hooksNotifier) DispatchTurnStart(
	ctx context.Context,
	payload hookspkg.TurnStartPayload,
) (hookspkg.TurnStartPayload, error) {
	return dispatchRuntime(
		ctx,
		n,
		hookspkg.HookTurnStart,
		payload,
		hookRuntime.DispatchTurnStart,
	)
}

func (n *hooksNotifier) DispatchTurnEnd(
	ctx context.Context,
	payload hookspkg.TurnEndPayload,
) (hookspkg.TurnEndPayload, error) {
	return dispatchRuntime(
		ctx,
		n,
		hookspkg.HookTurnEnd,
		payload,
		hookRuntime.DispatchTurnEnd,
	)
}

func (n *hooksNotifier) DispatchMessageStart(
	ctx context.Context,
	payload hookspkg.MessageStartPayload,
) (hookspkg.MessageStartPayload, error) {
	return dispatchRuntime(
		ctx,
		n,
		hookspkg.HookMessageStart,
		payload,
		hookRuntime.DispatchMessageStart,
	)
}

func (n *hooksNotifier) DispatchMessageDelta(
	ctx context.Context,
	payload hookspkg.MessageDeltaPayload,
) (hookspkg.MessageDeltaPayload, error) {
	return dispatchRuntime(
		ctx,
		n,
		hookspkg.HookMessageDelta,
		payload,
		hookRuntime.DispatchMessageDelta,
	)
}

func (n *hooksNotifier) DispatchMessageEnd(
	ctx context.Context,
	payload hookspkg.MessageEndPayload,
) (hookspkg.MessageEndPayload, error) {
	return dispatchRuntime(
		ctx,
		n,
		hookspkg.HookMessageEnd,
		payload,
		hookRuntime.DispatchMessageEnd,
	)
}

func (n *hooksNotifier) DispatchContextPreCompact(
	ctx context.Context,
	payload hookspkg.ContextPreCompactPayload,
) (hookspkg.ContextPreCompactPayload, error) {
	return dispatchRuntime(
		ctx,
		n,
		hookspkg.HookContextPreCompact,
		payload,
		hookRuntime.DispatchContextPreCompact,
	)
}

func (n *hooksNotifier) DispatchContextPostCompact(
	ctx context.Context,
	payload hookspkg.ContextPostCompactPayload,
) (hookspkg.ContextPostCompactPayload, error) {
	return dispatchRuntime(
		ctx,
		n,
		hookspkg.HookContextPostCompact,
		payload,
		hookRuntime.DispatchContextPostCompact,
	)
}

func (n *hooksNotifier) DispatchCoordinatorPreSpawn(
	ctx context.Context,
	payload hookspkg.CoordinatorPreSpawnPayload,
) (hookspkg.CoordinatorPreSpawnPayload, error) {
	return dispatchRuntime(
		ctx,
		n,
		hookspkg.HookCoordinatorPreSpawn,
		payload,
		hookRuntime.DispatchCoordinatorPreSpawn,
	)
}

func (n *hooksNotifier) DispatchCoordinatorSpawned(
	ctx context.Context,
	payload hookspkg.CoordinatorSpawnedPayload,
) (hookspkg.CoordinatorSpawnedPayload, error) {
	return dispatchRuntime(
		ctx,
		n,
		hookspkg.HookCoordinatorSpawned,
		payload,
		hookRuntime.DispatchCoordinatorSpawned,
	)
}

func (n *hooksNotifier) DispatchCoordinatorDecision(
	ctx context.Context,
	payload hookspkg.CoordinatorDecisionPayload,
) (hookspkg.CoordinatorDecisionPayload, error) {
	return dispatchRuntime(
		ctx,
		n,
		hookspkg.HookCoordinatorDecision,
		payload,
		hookRuntime.DispatchCoordinatorDecision,
	)
}

func (n *hooksNotifier) DispatchCoordinatorStopped(
	ctx context.Context,
	payload hookspkg.CoordinatorStoppedPayload,
) (hookspkg.CoordinatorStoppedPayload, error) {
	return dispatchRuntime(
		ctx,
		n,
		hookspkg.HookCoordinatorStopped,
		payload,
		hookRuntime.DispatchCoordinatorStopped,
	)
}

func (n *hooksNotifier) DispatchCoordinatorFailed(
	ctx context.Context,
	payload hookspkg.CoordinatorFailedPayload,
) (hookspkg.CoordinatorFailedPayload, error) {
	return dispatchRuntime(
		ctx,
		n,
		hookspkg.HookCoordinatorFailed,
		payload,
		hookRuntime.DispatchCoordinatorFailed,
	)
}

func (n *hooksNotifier) DispatchTaskRunEnqueued(
	ctx context.Context,
	payload hookspkg.TaskRunEnqueuedPayload,
) (hookspkg.TaskRunEnqueuedPayload, error) {
	result, err := dispatchRuntime(
		ctx,
		n,
		hookspkg.HookTaskRunEnqueued,
		payload,
		hookRuntime.DispatchTaskRunEnqueued,
	)
	if err != nil {
		return result, err
	}
	for _, observer := range n.taskRunEnqueuedObservers() {
		observer.OnTaskRunEnqueued(ctx, result)
	}
	return result, nil
}

func (n *hooksNotifier) DispatchTaskRunPreClaim(
	ctx context.Context,
	payload hookspkg.TaskRunPreClaimPayload,
) (hookspkg.TaskRunPreClaimPayload, error) {
	return dispatchRuntime(
		ctx,
		n,
		hookspkg.HookTaskRunPreClaim,
		payload,
		hookRuntime.DispatchTaskRunPreClaim,
	)
}

func (n *hooksNotifier) DispatchTaskRunPostClaim(
	ctx context.Context,
	payload hookspkg.TaskRunPostClaimPayload,
) (hookspkg.TaskRunPostClaimPayload, error) {
	return dispatchRuntime(
		ctx,
		n,
		hookspkg.HookTaskRunPostClaim,
		payload,
		hookRuntime.DispatchTaskRunPostClaim,
	)
}

func (n *hooksNotifier) DispatchTaskRunLeaseRecovered(
	ctx context.Context,
	payload hookspkg.TaskRunLeaseRecoveredPayload,
) (hookspkg.TaskRunLeaseRecoveredPayload, error) {
	return dispatchRuntime(
		ctx,
		n,
		hookspkg.HookTaskRunLeaseRecovered,
		payload,
		hookRuntime.DispatchTaskRunLeaseRecovered,
	)
}

func (n *hooksNotifier) DispatchTaskRunLeaseExtended(
	ctx context.Context,
	payload hookspkg.TaskRunLeaseExtendedPayload,
) (hookspkg.TaskRunLeaseExtendedPayload, error) {
	return dispatchRuntime(
		ctx,
		n,
		hookspkg.HookTaskRunLeaseExtended,
		payload,
		hookRuntime.DispatchTaskRunLeaseExtended,
	)
}

func (n *hooksNotifier) DispatchTaskRunLeaseExpired(
	ctx context.Context,
	payload hookspkg.TaskRunLeaseExpiredPayload,
) (hookspkg.TaskRunLeaseExpiredPayload, error) {
	return dispatchRuntime(
		ctx,
		n,
		hookspkg.HookTaskRunLeaseExpired,
		payload,
		hookRuntime.DispatchTaskRunLeaseExpired,
	)
}

func (n *hooksNotifier) DispatchTaskRunReleased(
	ctx context.Context,
	payload hookspkg.TaskRunReleasedPayload,
) (hookspkg.TaskRunReleasedPayload, error) {
	return dispatchRuntime(
		ctx,
		n,
		hookspkg.HookTaskRunReleased,
		payload,
		hookRuntime.DispatchTaskRunReleased,
	)
}

func (n *hooksNotifier) DispatchTaskRunCompleted(
	ctx context.Context,
	payload hookspkg.TaskRunCompletedPayload,
) (hookspkg.TaskRunCompletedPayload, error) {
	return dispatchRuntime(
		ctx,
		n,
		hookspkg.HookTaskRunCompleted,
		payload,
		hookRuntime.DispatchTaskRunCompleted,
	)
}

func (n *hooksNotifier) DispatchTaskRunFailed(
	ctx context.Context,
	payload hookspkg.TaskRunFailedPayload,
) (hookspkg.TaskRunFailedPayload, error) {
	return dispatchRuntime(
		ctx,
		n,
		hookspkg.HookTaskRunFailed,
		payload,
		hookRuntime.DispatchTaskRunFailed,
	)
}

func (n *hooksNotifier) DispatchSpawnPreCreate(
	ctx context.Context,
	payload hookspkg.SpawnPreCreatePayload,
) (hookspkg.SpawnPreCreatePayload, error) {
	return dispatchRuntime(
		ctx,
		n,
		hookspkg.HookSpawnPreCreate,
		payload,
		hookRuntime.DispatchSpawnPreCreate,
	)
}

func (n *hooksNotifier) DispatchSpawnCreated(
	ctx context.Context,
	payload hookspkg.SpawnCreatedPayload,
) (hookspkg.SpawnCreatedPayload, error) {
	return dispatchRuntime(
		ctx,
		n,
		hookspkg.HookSpawnCreated,
		payload,
		hookRuntime.DispatchSpawnCreated,
	)
}

func (n *hooksNotifier) DispatchSpawnParentStopped(
	ctx context.Context,
	payload hookspkg.SpawnParentStoppedPayload,
) (hookspkg.SpawnParentStoppedPayload, error) {
	return dispatchRuntime(
		ctx,
		n,
		hookspkg.HookSpawnParentStopped,
		payload,
		hookRuntime.DispatchSpawnParentStopped,
	)
}

func (n *hooksNotifier) DispatchSpawnTTLExpired(
	ctx context.Context,
	payload hookspkg.SpawnTTLExpiredPayload,
) (hookspkg.SpawnTTLExpiredPayload, error) {
	return dispatchRuntime(
		ctx,
		n,
		hookspkg.HookSpawnTTLExpired,
		payload,
		hookRuntime.DispatchSpawnTTLExpired,
	)
}

func (n *hooksNotifier) DispatchSpawnReaped(
	ctx context.Context,
	payload hookspkg.SpawnReapedPayload,
) (hookspkg.SpawnReapedPayload, error) {
	return dispatchRuntime(
		ctx,
		n,
		hookspkg.HookSpawnReaped,
		payload,
		hookRuntime.DispatchSpawnReaped,
	)
}

func (n *hooksNotifier) DispatchAgentSoulSnapshotResolved(
	ctx context.Context,
	payload hookspkg.AgentSoulSnapshotResolvedPayload,
) (hookspkg.AgentSoulSnapshotResolvedPayload, error) {
	return dispatchRuntime(
		ctx,
		n,
		hookspkg.HookAgentSoulSnapshotResolved,
		payload,
		hookRuntime.DispatchAgentSoulSnapshotResolved,
	)
}

func (n *hooksNotifier) DispatchAgentSoulMutationAfter(
	ctx context.Context,
	payload hookspkg.AgentSoulMutationAfterPayload,
) (hookspkg.AgentSoulMutationAfterPayload, error) {
	return dispatchRuntime(
		ctx,
		n,
		hookspkg.HookAgentSoulMutationAfter,
		payload,
		hookRuntime.DispatchAgentSoulMutationAfter,
	)
}

func (n *hooksNotifier) DispatchAgentHeartbeatPolicyResolved(
	ctx context.Context,
	payload hookspkg.AgentHeartbeatPolicyResolvedPayload,
) (hookspkg.AgentHeartbeatPolicyResolvedPayload, error) {
	return dispatchRuntime(
		ctx,
		n,
		hookspkg.HookAgentHeartbeatPolicyResolved,
		payload,
		hookRuntime.DispatchAgentHeartbeatPolicyResolved,
	)
}

func (n *hooksNotifier) DispatchAgentHeartbeatWakeBefore(
	ctx context.Context,
	payload hookspkg.AgentHeartbeatWakeBeforePayload,
) (hookspkg.AgentHeartbeatWakeBeforePayload, error) {
	return dispatchRuntime(
		ctx,
		n,
		hookspkg.HookAgentHeartbeatWakeBefore,
		payload,
		hookRuntime.DispatchAgentHeartbeatWakeBefore,
	)
}

func (n *hooksNotifier) DispatchAgentHeartbeatWakeAfter(
	ctx context.Context,
	payload hookspkg.AgentHeartbeatWakeAfterPayload,
) (hookspkg.AgentHeartbeatWakeAfterPayload, error) {
	return dispatchRuntime(
		ctx,
		n,
		hookspkg.HookAgentHeartbeatWakeAfter,
		payload,
		hookRuntime.DispatchAgentHeartbeatWakeAfter,
	)
}

func (n *hooksNotifier) DispatchSessionHealthUpdateAfter(
	ctx context.Context,
	payload hookspkg.SessionHealthUpdateAfterPayload,
) (hookspkg.SessionHealthUpdateAfterPayload, error) {
	return dispatchRuntime(
		ctx,
		n,
		hookspkg.HookSessionHealthUpdateAfter,
		payload,
		hookRuntime.DispatchSessionHealthUpdateAfter,
	)
}

func (n *hooksNotifier) OnAgentEvent(ctx context.Context, sessionID string, event any) {
	n.dispatchAgentEvent(ctx, hookspkg.SessionContext{SessionID: strings.TrimSpace(sessionID)}, event)
}

func (n *hooksNotifier) OnAgentEventForSession(ctx context.Context, sess *session.Session, event any) {
	n.dispatchAgentEvent(ctx, hookSessionContext(sess), event)
}

func (n *hooksNotifier) OnSandboxLifecycleEvent(ctx context.Context, event session.SandboxLifecycleEvent) {
	_, agentEventNotify := n.runtime()
	if notifier, ok := agentEventNotify.(session.SandboxLifecycleNotifier); ok {
		notifier.OnSandboxLifecycleEvent(ctx, event)
	}
}

func (n *hooksNotifier) dispatchAgentEvent(ctx context.Context, sessionCtx hookspkg.SessionContext, event any) {
	hooks, agentEventNotify := n.runtime()
	if agentEventNotify != nil {
		agentEventNotify.OnAgentEvent(ctx, sessionCtx.SessionID, event)
	}
	if hooks != nil {
		dispatchACPAgentHookEvent(ctx, n.logger, hooks, sessionCtx, event, n.timestamp())
	}
}

func (n *hooksNotifier) runtime() (hookRuntime, session.Notifier) {
	n.mu.RLock()
	defer n.mu.RUnlock()

	return n.hooks, n.agentEventNotify
}

func (n *hooksNotifier) timestamp() time.Time {
	if n == nil || n.now == nil {
		return time.Now().UTC()
	}
	return n.now().UTC()
}

type runtimeDispatchFunc[P any] func(hookRuntime, context.Context, P) (P, error)

func dispatchRuntime[P any](
	ctx context.Context,
	n *hooksNotifier,
	event hookspkg.HookEvent,
	payload P,
	dispatch runtimeDispatchFunc[P],
) (P, error) {
	hooks, _ := n.runtime()
	if hooks == nil {
		return payload, nil
	}
	if ctx == nil {
		return payload, fmt.Errorf("daemon: dispatch %s requires a non-nil context", event)
	}
	return dispatch(hooks, ctx, payload)
}

func hookSessionLifecyclePayload(
	sess *session.Session,
	event hookspkg.HookEvent,
	timestamp time.Time,
) hookspkg.SessionLifecyclePayload {
	return hookspkg.SessionLifecyclePayload{
		PayloadBase: hookspkg.PayloadBase{
			Event:     event,
			Timestamp: timestamp,
		},
		SessionContext: hookSessionContext(sess),
	}
}

func hookSessionContext(sess *session.Session) hookspkg.SessionContext {
	if sess == nil {
		return hookspkg.SessionContext{}
	}

	info := sess.Info()
	if info == nil {
		return hookspkg.SessionContext{}
	}

	return hookspkg.SessionContext{
		SessionID:    strings.TrimSpace(info.ID),
		SessionName:  strings.TrimSpace(info.Name),
		SessionType:  string(info.Type),
		AgentName:    strings.TrimSpace(info.AgentName),
		WorkspaceID:  strings.TrimSpace(info.WorkspaceID),
		Workspace:    strings.TrimSpace(info.Workspace),
		ACPSessionID: strings.TrimSpace(info.ACPSessionID),
		State:        string(info.State),
		SessionSoulContext: hookSessionSoulContext(
			info.SoulSnapshotID,
			info.SoulDigest,
		),
		CreatedAt: info.CreatedAt,
		UpdatedAt: info.UpdatedAt,
	}
}

func hookSessionSoulContext(snapshotID string, digest string) *hookspkg.SessionSoulContext {
	trimmedSnapshotID := strings.TrimSpace(snapshotID)
	trimmedDigest := strings.TrimSpace(digest)
	if trimmedSnapshotID == "" && trimmedDigest == "" {
		return nil
	}
	return &hookspkg.SessionSoulContext{
		SoulSnapshotID: trimmedSnapshotID,
		SoulDigest:     trimmedDigest,
	}
}

func sessionFromHookPayload(payload hookspkg.SessionLifecyclePayload) *session.Session {
	return &session.Session{
		ID:           strings.TrimSpace(payload.SessionID),
		Name:         strings.TrimSpace(payload.SessionName),
		AgentName:    strings.TrimSpace(payload.AgentName),
		WorkspaceID:  strings.TrimSpace(payload.WorkspaceID),
		Workspace:    strings.TrimSpace(payload.Workspace),
		Type:         session.Type(strings.TrimSpace(payload.SessionType)),
		State:        session.State(strings.TrimSpace(payload.State)),
		ACPSessionID: strings.TrimSpace(payload.ACPSessionID),
		CreatedAt:    payload.CreatedAt,
		UpdatedAt:    payload.UpdatedAt,
	}
}

func observeSessionCreateExecutor(observer sessionLifecycleObserver) hookspkg.Executor {
	return hookspkg.NewTypedNativeExecutor(
		func(
			ctx context.Context,
			_ hookspkg.RegisteredHook,
			payload hookspkg.SessionLifecyclePayload,
		) (hookspkg.SessionPostCreatePatch, error) {
			observer.OnSessionCreated(ctx, sessionFromHookPayload(payload))
			return hookspkg.SessionPostCreatePatch{}, nil
		},
	)
}

func observeSessionStopExecutor(observer sessionLifecycleObserver) hookspkg.Executor {
	return hookspkg.NewTypedNativeExecutor(
		func(
			ctx context.Context,
			_ hookspkg.RegisteredHook,
			payload hookspkg.SessionLifecyclePayload,
		) (hookspkg.SessionPostStopPatch, error) {
			observer.OnSessionStopped(ctx, sessionFromHookPayload(payload))
			return hookspkg.SessionPostStopPatch{}, nil
		},
	)
}

func dreamSessionStopExecutor(dreamRuntime dreamCheckEnqueuer) hookspkg.Executor {
	return hookspkg.NewTypedNativeExecutor(
		func(
			_ context.Context,
			_ hookspkg.RegisteredHook,
			payload hookspkg.SessionLifecyclePayload,
		) (hookspkg.SessionPostStopPatch, error) {
			if strings.TrimSpace(payload.WorkspaceID) == "" ||
				session.Type(strings.TrimSpace(payload.SessionType)) == session.SessionTypeDream {
				return hookspkg.SessionPostStopPatch{}, nil
			}

			dreamRuntime.EnqueueCheck("session_stop", strings.TrimSpace(payload.WorkspaceID))
			return hookspkg.SessionPostStopPatch{}, nil
		},
	)
}

func daemonNativeHooks(
	observer sessionLifecycleObserver,
	dreamRuntime dreamCheckEnqueuer,
) ([]hookspkg.HookDecl, map[string]hookspkg.Executor) {
	decls := make([]hookspkg.HookDecl, 0, 3)
	executors := make(map[string]hookspkg.Executor, 3)

	if observer != nil {
		const (
			createName = "daemon.observe.session_post_create"
			stopName   = "daemon.observe.session_post_stop"
		)

		decls = append(decls,
			hookspkg.HookDecl{
				Name:         createName,
				Event:        hookspkg.HookSessionPostCreate,
				Mode:         hookspkg.HookModeSync,
				Priority:     1000,
				PrioritySet:  true,
				ExecutorKind: hookspkg.HookExecutorNative,
			},
			hookspkg.HookDecl{
				Name:         stopName,
				Event:        hookspkg.HookSessionPostStop,
				Mode:         hookspkg.HookModeSync,
				Priority:     1000,
				PrioritySet:  true,
				ExecutorKind: hookspkg.HookExecutorNative,
			},
		)
		executors[createName] = observeSessionCreateExecutor(observer)
		executors[stopName] = observeSessionStopExecutor(observer)
	}

	if dreamRuntime != nil {
		const dreamName = "daemon.dream.session_post_stop"

		decls = append(decls, hookspkg.HookDecl{
			Name:         dreamName,
			Event:        hookspkg.HookSessionPostStop,
			Mode:         hookspkg.HookModeSync,
			Priority:     900,
			PrioritySet:  true,
			ExecutorKind: hookspkg.HookExecutorNative,
		})
		executors[dreamName] = dreamSessionStopExecutor(dreamRuntime)
	}

	return decls, executors
}

func daemonExecutorResolver(nativeExecutors map[string]hookspkg.Executor) hookspkg.ExecutorResolver {
	return daemonExecutorResolverWithSecrets(nativeExecutors, nil)
}

func daemonExecutorResolverWithSecrets(
	nativeExecutors map[string]hookspkg.Executor,
	secretResolver hookspkg.SecretRefResolver,
	registries ...*toolruntime.Registry,
) hookspkg.ExecutorResolver {
	var registry *toolruntime.Registry
	if len(registries) > 0 {
		registry = registries[0]
	}
	return func(decl hookspkg.HookDecl) (hookspkg.Executor, error) {
		if decl.ExecutorKind == hookspkg.HookExecutorNative {
			executor := nativeExecutors[strings.TrimSpace(decl.Name)]
			if executor == nil {
				return nil, fmt.Errorf("daemon: missing native hook executor for %q", decl.Name)
			}
			return executor, nil
		}
		return defaultDaemonExecutorResolverWithRegistry(decl, secretResolver, registry)
	}
}

func defaultDaemonExecutorResolver(decl hookspkg.HookDecl) (hookspkg.Executor, error) {
	return defaultDaemonExecutorResolverWithRegistry(decl, nil, nil)
}

func defaultDaemonExecutorResolverWithRegistry(
	decl hookspkg.HookDecl,
	secretResolver hookspkg.SecretRefResolver,
	registry *toolruntime.Registry,
) (hookspkg.Executor, error) {
	switch decl.ExecutorKind {
	case hookspkg.HookExecutorSubprocess:
		opts := []hookspkg.SubprocessExecutorOption{
			hookspkg.WithSubprocessEnv(decl.Env),
			hookspkg.WithSubprocessSecretEnv(decl.SecretEnv, secretResolver),
		}
		if registry != nil {
			opts = append(opts, hookspkg.WithSubprocessProcessRegistry(registry))
		}
		if dir := strings.TrimSpace(decl.WorkingDir); dir != "" {
			opts = append(opts, hookspkg.WithSubprocessDir(dir))
		} else if root := strings.TrimSpace(decl.Matcher.WorkspaceRoot); root != "" {
			opts = append(opts, hookspkg.WithSubprocessDir(root))
		}
		return hookspkg.NewSubprocessExecutor(
			decl.Command,
			decl.Args,
			opts...,
		), nil
	case hookspkg.HookExecutorWASM:
		return &hookspkg.WasmExecutor{}, nil
	case hookspkg.HookExecutorNative:
		return nil, fmt.Errorf("daemon: native executor for hook %q requires an explicit binding", decl.Name)
	default:
		return nil, fmt.Errorf("daemon: unsupported executor kind %q for hook %q", decl.ExecutorKind, decl.Name)
	}
}

func chainDeclarationProviders(providers ...hookspkg.DeclarationProvider) hookspkg.DeclarationProvider {
	return func(ctx context.Context) ([]hookspkg.HookDecl, error) {
		chained := make([]hookspkg.HookDecl, 0, len(providers))
		for idx, provider := range providers {
			if provider == nil {
				continue
			}

			decls, err := provider(ctx)
			if err != nil {
				return nil, fmt.Errorf("daemon: load hook declarations from provider %d: %w", idx+1, err)
			}
			chained = append(chained, decls...)
		}
		return chained, nil
	}
}

func extensionDeclarationProvider(getRuntime func() extensionRuntime) hookspkg.DeclarationProvider {
	return func(ctx context.Context) ([]hookspkg.HookDecl, error) {
		if getRuntime == nil {
			return nil, nil
		}

		runtime := getRuntime()
		if runtime == nil {
			return nil, nil
		}
		decls, err := runtime.HookDeclarations(ctx)
		if err != nil {
			return nil, fmt.Errorf("daemon: load hook declarations from extension runtime: %w", err)
		}
		return decls, nil
	}
}

func configDeclarationProvider(
	registry Registry,
	workspaceResolver workspacepkg.RuntimeResolver,
	logger *slog.Logger,
) hookspkg.DeclarationProvider {
	if logger == nil {
		logger = slog.Default()
	}
	return func(ctx context.Context) ([]hookspkg.HookDecl, error) {
		decls, err := workspaceHookDeclarations(ctx, registry, workspaceResolver, logger)
		if err != nil {
			return nil, err
		}
		return filterHookDeclsBySource(decls, hookspkg.HookSourceConfig), nil
	}
}

func agentDeclarationProvider(
	registry Registry,
	workspaceResolver workspacepkg.RuntimeResolver,
	logger *slog.Logger,
) hookspkg.DeclarationProvider {
	if logger == nil {
		logger = slog.Default()
	}
	return func(ctx context.Context) ([]hookspkg.HookDecl, error) {
		decls, err := workspaceHookDeclarations(ctx, registry, workspaceResolver, logger)
		if err != nil {
			return nil, err
		}
		return filterHookDeclsBySource(decls, hookspkg.HookSourceAgentDefinition), nil
	}
}

func skillDeclarationProvider(
	skillsRegistry *skills.Registry,
	registry Registry,
	workspaceResolver workspacepkg.RuntimeResolver,
	allowedMarketplaceHooks []string,
	logger *slog.Logger,
) hookspkg.DeclarationProvider {
	if logger == nil {
		logger = slog.Default()
	}
	allowed := marketplaceHookAllowlist(allowedMarketplaceHooks)

	return func(ctx context.Context) ([]hookspkg.HookDecl, error) {
		if skillsRegistry == nil || registry == nil || workspaceResolver == nil {
			return nil, nil
		}

		workspaces, err := registeredWorkspaces(ctx, registry, workspaceResolver, logger)
		if err != nil {
			return nil, err
		}

		decls := make([]hookspkg.HookDecl, 0, len(workspaces))
		for idx := range workspaces {
			resolved := &workspaces[idx]
			activeSkills, err := skillsRegistry.ForWorkspace(ctx, resolved)
			if err != nil {
				return nil, fmt.Errorf("daemon: resolve active skills for workspace %q: %w", resolved.ID, err)
			}

			for _, skill := range activeSkills {
				if !marketplaceHookAllowed(skill, allowed) {
					logger.Warn(
						"daemon: blocked hook",
						"skill_name", skill.Meta.Name,
						"workspace_id", resolved.ID,
						"source", skills.SkillSourceName(skill.Source),
					)
					continue
				}
				decls = append(decls, scopeWorkspaceHookDecls(skill.Hooks, resolved)...)
			}
		}

		return decls, nil
	}
}

func workspaceHookDeclarations(
	ctx context.Context,
	registry Registry,
	workspaceResolver workspacepkg.RuntimeResolver,
	logger *slog.Logger,
) ([]hookspkg.HookDecl, error) {
	workspaces, err := registeredWorkspaces(ctx, registry, workspaceResolver, logger)
	if err != nil {
		return nil, err
	}

	decls := make([]hookspkg.HookDecl, 0, len(workspaces))
	for idx := range workspaces {
		resolved := &workspaces[idx]
		workspaceDecls, err := aghconfig.HookDeclarations(resolved.Config.Hooks, resolved.Agents)
		if err != nil {
			return nil, fmt.Errorf("daemon: load hook declarations for workspace %q: %w", resolved.ID, err)
		}
		decls = append(decls, scopeWorkspaceHookDecls(workspaceDecls, resolved)...)
	}

	return decls, nil
}

func registeredWorkspaces(
	ctx context.Context,
	registry Registry,
	workspaceResolver workspacepkg.RuntimeResolver,
	logger *slog.Logger,
) ([]workspacepkg.ResolvedWorkspace, error) {
	if registry == nil || workspaceResolver == nil {
		return nil, nil
	}

	workspaces, err := registry.ListWorkspaces(ctx)
	if err != nil {
		return nil, fmt.Errorf("daemon: list workspaces for hooks rebuild: %w", err)
	}
	slices.SortFunc(workspaces, func(left, right workspacepkg.Workspace) int {
		return strings.Compare(strings.TrimSpace(left.ID), strings.TrimSpace(right.ID))
	})

	resolvedWorkspaces := make([]workspacepkg.ResolvedWorkspace, 0, len(workspaces))
	for _, workspace := range workspaces {
		resolved, err := workspaceResolver.Resolve(ctx, workspace.ID)
		switch {
		case err == nil:
			resolvedWorkspaces = append(resolvedWorkspaces, resolved)
		case errors.Is(err, workspacepkg.ErrWorkspaceNotFound), errors.Is(err, workspacepkg.ErrWorkspaceRootMissing):
			if logger != nil {
				logger.Warn(
					"daemon: skipped workspace while rebuilding hooks",
					"workspace_id", workspace.ID,
					"workspace_root", workspace.RootDir,
					"error", err,
				)
			}
		default:
			return nil, fmt.Errorf("daemon: resolve workspace %q for hooks rebuild: %w", workspace.ID, err)
		}
	}

	return resolvedWorkspaces, nil
}

func filterHookDeclsBySource(decls []hookspkg.HookDecl, source hookspkg.HookSource) []hookspkg.HookDecl {
	filtered := make([]hookspkg.HookDecl, 0, len(decls))
	for _, decl := range decls {
		if decl.Source != source {
			continue
		}
		filtered = append(filtered, cloneDaemonHookDecl(decl))
	}
	return filtered
}

func scopeWorkspaceHookDecls(
	decls []hookspkg.HookDecl,
	resolved *workspacepkg.ResolvedWorkspace,
) []hookspkg.HookDecl {
	scoped := make([]hookspkg.HookDecl, 0, len(decls))
	for _, decl := range decls {
		cloned := cloneDaemonHookDecl(decl)
		if resolved != nil {
			if strings.TrimSpace(cloned.WorkingDir) == "" {
				cloned.WorkingDir = strings.TrimSpace(resolved.RootDir)
			}
			if hookspkg.MatcherFieldAllowedForEvent(cloned.Event, "workspace_id") {
				cloned.Matcher.WorkspaceID = strings.TrimSpace(resolved.ID)
			}
			if hookspkg.MatcherFieldAllowedForEvent(cloned.Event, "workspace_root") {
				cloned.Matcher.WorkspaceRoot = strings.TrimSpace(resolved.RootDir)
			}
		}
		scoped = append(scoped, cloned)
	}
	return scoped
}

func cloneDaemonHookDecl(src hookspkg.HookDecl) hookspkg.HookDecl {
	cloned := src
	cloned.Args = append([]string(nil), src.Args...)
	cloned.Env = cloneStringMap(src.Env)
	cloned.SecretEnv = cloneStringMap(src.SecretEnv)
	cloned.Metadata = cloneStringMap(src.Metadata)
	if src.Matcher.ToolReadOnly != nil {
		value := *src.Matcher.ToolReadOnly
		cloned.Matcher.ToolReadOnly = &value
	}
	return cloned
}

func cloneStringMap(src map[string]string) map[string]string {
	if len(src) == 0 {
		return nil
	}

	cloned := make(map[string]string, len(src))
	maps.Copy(cloned, src)
	return cloned
}

func marketplaceHookAllowlist(values []string) map[string]struct{} {
	allowed := make(map[string]struct{}, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		allowed[trimmed] = struct{}{}
	}

	return allowed
}

func marketplaceHookAllowed(skill *skills.Skill, allowedMarketplaceHooks map[string]struct{}) bool {
	if skill == nil {
		return false
	}

	switch skill.Source {
	case skills.SourceBundled, skills.SourceUser, skills.SourceAdditional, skills.SourceWorkspace:
		return true
	case skills.SourceMarketplace:
		for _, key := range marketplaceHookConsentKeys(skill) {
			if _, ok := allowedMarketplaceHooks[key]; ok {
				return true
			}
		}
		return false
	default:
		return false
	}
}

func marketplaceHookConsentKeys(skill *skills.Skill) []string {
	if skill == nil || skill.Provenance == nil {
		return nil
	}

	keys := make([]string, 0, 3)
	if slug := strings.TrimSpace(skill.Provenance.Slug); slug != "" {
		keys = append(keys, slug)
		if registry := strings.TrimSpace(skill.Provenance.Registry); registry != "" {
			keys = append(keys, registry+":"+slug)
		}
	}
	if hash := strings.TrimSpace(skill.Provenance.Hash); hash != "" {
		keys = append(keys, hash)
	}

	return keys
}
