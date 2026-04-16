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
	DispatchEnvironmentPrepare(
		context.Context,
		hookspkg.EnvironmentPreparePayload,
	) (hookspkg.EnvironmentPreparePayload, error)
	DispatchEnvironmentReady(
		context.Context,
		hookspkg.EnvironmentReadyPayload,
	) (hookspkg.EnvironmentReadyPayload, error)
	DispatchEnvironmentSyncBefore(
		context.Context,
		hookspkg.EnvironmentSyncBeforePayload,
	) (hookspkg.EnvironmentSyncBeforePayload, error)
	DispatchEnvironmentSyncAfter(
		context.Context,
		hookspkg.EnvironmentSyncAfterPayload,
	) (hookspkg.EnvironmentSyncAfterPayload, error)
	DispatchEnvironmentStop(context.Context, hookspkg.EnvironmentStopPayload) (hookspkg.EnvironmentStopPayload, error)
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
}

type sessionLifecycleObserver interface {
	OnSessionCreated(context.Context, *session.Session)
	OnSessionStopped(context.Context, *session.Session)
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

	logger           *slog.Logger
	now              func() time.Time
	hooks            hookRuntime
	agentEventNotify session.Notifier
}

var _ session.Notifier = (*hooksNotifier)(nil)
var _ session.LifecycleHooks = (*hooksNotifier)(nil)
var _ session.EnvironmentHooks = (*hooksNotifier)(nil)
var _ session.PromptHooks = (*hooksNotifier)(nil)
var _ session.EventHooks = (*hooksNotifier)(nil)
var _ session.AgentHooks = (*hooksNotifier)(nil)
var _ session.ConversationHooks = (*hooksNotifier)(nil)
var _ session.CompactionHooks = (*hooksNotifier)(nil)
var _ session.AgentEventNotifier = (*hooksNotifier)(nil)
var _ session.EnvironmentLifecycleNotifier = (*hooksNotifier)(nil)

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

// OnSessionCreated is a no-op; lifecycle observation is handled via hook dispatch.
func (n *hooksNotifier) OnSessionCreated(_ context.Context, _ *session.Session) {
}

// OnSessionStopped is a no-op; lifecycle observation is handled via hook dispatch.
func (n *hooksNotifier) OnSessionStopped(_ context.Context, _ *session.Session) {
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

func (n *hooksNotifier) DispatchEnvironmentPrepare(
	ctx context.Context,
	payload hookspkg.EnvironmentPreparePayload,
) (hookspkg.EnvironmentPreparePayload, error) {
	return dispatchRuntime(
		ctx,
		n,
		hookspkg.HookEnvironmentPrepare,
		payload,
		hookRuntime.DispatchEnvironmentPrepare,
	)
}

func (n *hooksNotifier) DispatchEnvironmentReady(
	ctx context.Context,
	payload hookspkg.EnvironmentReadyPayload,
) (hookspkg.EnvironmentReadyPayload, error) {
	return dispatchRuntime(
		ctx,
		n,
		hookspkg.HookEnvironmentReady,
		payload,
		hookRuntime.DispatchEnvironmentReady,
	)
}

func (n *hooksNotifier) DispatchEnvironmentSyncBefore(
	ctx context.Context,
	payload hookspkg.EnvironmentSyncBeforePayload,
) (hookspkg.EnvironmentSyncBeforePayload, error) {
	return dispatchRuntime(
		ctx,
		n,
		hookspkg.HookEnvironmentSyncBefore,
		payload,
		hookRuntime.DispatchEnvironmentSyncBefore,
	)
}

func (n *hooksNotifier) DispatchEnvironmentSyncAfter(
	ctx context.Context,
	payload hookspkg.EnvironmentSyncAfterPayload,
) (hookspkg.EnvironmentSyncAfterPayload, error) {
	return dispatchRuntime(
		ctx,
		n,
		hookspkg.HookEnvironmentSyncAfter,
		payload,
		hookRuntime.DispatchEnvironmentSyncAfter,
	)
}

func (n *hooksNotifier) DispatchEnvironmentStop(
	ctx context.Context,
	payload hookspkg.EnvironmentStopPayload,
) (hookspkg.EnvironmentStopPayload, error) {
	return dispatchRuntime(
		ctx,
		n,
		hookspkg.HookEnvironmentStop,
		payload,
		hookRuntime.DispatchEnvironmentStop,
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

func (n *hooksNotifier) OnAgentEvent(ctx context.Context, sessionID string, event any) {
	n.dispatchAgentEvent(ctx, hookspkg.SessionContext{SessionID: strings.TrimSpace(sessionID)}, event)
}

func (n *hooksNotifier) OnAgentEventForSession(ctx context.Context, sess *session.Session, event any) {
	n.dispatchAgentEvent(ctx, hookSessionContext(sess), event)
}

func (n *hooksNotifier) OnEnvironmentLifecycleEvent(ctx context.Context, event session.EnvironmentLifecycleEvent) {
	_, agentEventNotify := n.runtime()
	if notifier, ok := agentEventNotify.(session.EnvironmentLifecycleNotifier); ok {
		notifier.OnEnvironmentLifecycleEvent(ctx, event)
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
		CreatedAt:    info.CreatedAt,
		UpdatedAt:    info.UpdatedAt,
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
	return func(decl hookspkg.HookDecl) (hookspkg.Executor, error) {
		if decl.ExecutorKind == hookspkg.HookExecutorNative {
			executor := nativeExecutors[strings.TrimSpace(decl.Name)]
			if executor == nil {
				return nil, fmt.Errorf("daemon: missing native hook executor for %q", decl.Name)
			}
			return executor, nil
		}
		return defaultDaemonExecutorResolver(decl)
	}
}

func defaultDaemonExecutorResolver(decl hookspkg.HookDecl) (hookspkg.Executor, error) {
	switch decl.ExecutorKind {
	case hookspkg.HookExecutorSubprocess:
		opts := []hookspkg.SubprocessExecutorOption{
			hookspkg.WithSubprocessEnv(decl.Env),
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
			cloned.Matcher.WorkspaceID = strings.TrimSpace(resolved.ID)
			cloned.Matcher.WorkspaceRoot = strings.TrimSpace(resolved.RootDir)
		}
		scoped = append(scoped, cloned)
	}
	return scoped
}

func cloneDaemonHookDecl(src hookspkg.HookDecl) hookspkg.HookDecl {
	cloned := src
	cloned.Args = append([]string(nil), src.Args...)
	cloned.Env = cloneStringMap(src.Env)
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
