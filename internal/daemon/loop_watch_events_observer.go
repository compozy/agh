package daemon

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"strings"
	"sync"
	"time"

	eventspkg "github.com/compozy/agh/internal/events"
	hookspkg "github.com/compozy/agh/internal/hooks"
	looppkg "github.com/compozy/agh/internal/loop"
	"github.com/compozy/agh/internal/store"
	taskpkg "github.com/compozy/agh/internal/task"
)

const (
	loopWatchEventsMatchedEvent      = "loop.watch_events.matched"
	loopWatchEventsWakeEnqueuedEvent = "loop.watch_events.wake_enqueued"
	loopWatchEventsWakeErrorEvent    = "loop.watch_events.wake_error"

	loopWatchEventsObserverSessionID = "loop-watch-events-observer"
	loopWatchEventsObserverActorID   = "loop-watch-events-observer"
	loopWatchEventsDaemonAgentName   = "daemon"
	loopWatchEventsDoorbellReason    = "watch_events_doorbell"

	loopWatchEventsContentLoopNameField  = "loop_name"
	loopWatchEventsContentLoopRunIDField = "loop_run_id"
	// #nosec G101 -- event-summary field name, not credential material.
	loopWatchEventsContentIdempotencyField = "idempotency_key"
	loopWatchEventsContentWorkspaceIDField = hooksBridgeWorkspaceIDKey
	loopWatchEventsContentWakeRunIDField   = "wake_run_id"
	loopWatchEventsContentEventRunIDField  = "event_run_id"
	// #nosec G101 -- event-summary field name, not credential material.
	loopWatchEventsContentEventTaskIDField = "event_task_id"
	loopWatchEventsContentHookEventField   = "hook_event"
	loopWatchEventsContentAddedField       = "added"
)

type loopWatchEventsStore interface {
	ListParkedWatchEventSubscriptions(context.Context) ([]looppkg.ParkedWatchEventSubscription, error)
	EnqueueLoopCoordinatorWake(context.Context, string, string, taskpkg.Origin, time.Time) (taskpkg.Run, bool, error)
	WriteEventSummary(context.Context, store.EventSummary) error
}

type loopWatchEventsGapWakeStore interface {
	EnqueueWatchEventsGapWakes(context.Context, taskpkg.Origin, time.Time) ([]taskpkg.Run, error)
}

type loopWatchEventsObserver struct {
	store    loopWatchEventsStore
	backstop loopCoordinatorBackstopRunner
	actor    taskpkg.ActorContext
	now      func() time.Time
	compiler *watchEventsDoorbellCompiler

	mu     sync.RWMutex
	byKind map[hookspkg.HookEvent][]looppkg.ParkedWatchEventSubscription
}

var _ taskStatusChangedObserver = (*loopWatchEventsObserver)(nil)
var _ taskLifecycleWatchObserver = (*loopWatchEventsObserver)(nil)
var _ taskRunTerminalObserver = (*loopWatchEventsObserver)(nil)
var _ loopTerminalObserver = (*loopWatchEventsObserver)(nil)
var _ loopNodeTerminalObserver = (*loopWatchEventsObserver)(nil)
var _ automationRunWatchObserver = (*loopWatchEventsObserver)(nil)
var _ networkWatchObserver = (*loopWatchEventsObserver)(nil)

func newLoopWatchEventsObserver(
	store loopWatchEventsStore,
	backstop loopCoordinatorBackstopRunner,
	now func() time.Time,
) (*loopWatchEventsObserver, error) {
	if store == nil {
		return nil, fmt.Errorf("daemon: loop watch-events observer requires store")
	}
	if backstop == nil {
		return nil, fmt.Errorf("daemon: loop watch-events observer requires coordinator backstop")
	}
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}
	actor, err := taskpkg.DeriveDaemonActorContext(loopWatchEventsObserverActorID, "loop.watch_events")
	if err != nil {
		return nil, fmt.Errorf("daemon: derive loop watch-events actor: %w", err)
	}
	compiler, err := newWatchEventsDoorbellCompiler()
	if err != nil {
		return nil, err
	}
	return &loopWatchEventsObserver{
		store:    store,
		backstop: backstop,
		actor:    actor,
		now:      now,
		compiler: compiler,
		byKind:   map[hookspkg.HookEvent][]looppkg.ParkedWatchEventSubscription{},
	}, nil
}

func (o *loopWatchEventsObserver) Hydrate(ctx context.Context) error {
	parked, err := o.store.ListParkedWatchEventSubscriptions(ctx)
	if err != nil {
		return fmt.Errorf("daemon: hydrate loop watch-events observer: %w", err)
	}
	o.replaceIndex(parked)
	return nil
}

func (o *loopWatchEventsObserver) replaceIndex(entries []looppkg.ParkedWatchEventSubscription) {
	index := make(map[hookspkg.HookEvent][]looppkg.ParkedWatchEventSubscription)
	supported := looppkg.SupportedWatchEvents()
	for _, entry := range entries {
		cloned := cloneParkedWatchEventSubscription(entry)
		seen := map[hookspkg.HookEvent]struct{}{}
		for _, ref := range cloned.Subscriptions {
			kind := hookspkg.HookEvent(strings.TrimSpace(ref.Kind))
			if _, ok := supported[kind]; !ok {
				continue
			}
			if _, ok := seen[kind]; ok {
				continue
			}
			seen[kind] = struct{}{}
			index[kind] = append(index[kind], cloned)
		}
	}
	o.mu.Lock()
	o.byKind = index
	o.mu.Unlock()
}

func (o *loopWatchEventsObserver) OnTaskStatusChanged(
	ctx context.Context,
	payload hookspkg.TaskStatusChangedPayload,
) error {
	return o.matchAndWake(ctx, watchEventsTaskStatusChangedEvent(payload, o.now))
}

func (o *loopWatchEventsObserver) OnTaskBlocked(ctx context.Context, payload hookspkg.TaskBlockedPayload) error {
	return o.matchAndWake(ctx, watchEventsTaskBlockEvent(hookspkg.HookTaskBlocked, payload, o.now))
}

func (o *loopWatchEventsObserver) OnTaskUnblocked(ctx context.Context, payload hookspkg.TaskUnblockedPayload) error {
	return o.matchAndWake(ctx, watchEventsTaskBlockEvent(hookspkg.HookTaskUnblocked, payload, o.now))
}

func (o *loopWatchEventsObserver) OnTaskNeedsAttention(
	ctx context.Context,
	payload hookspkg.TaskNeedsAttentionPayload,
) error {
	return o.matchAndWake(ctx, watchEventsTaskAttentionEvent(hookspkg.HookTaskNeedsAttention, payload, o.now))
}

func (o *loopWatchEventsObserver) OnTaskRecovered(ctx context.Context, payload hookspkg.TaskRecoveredPayload) error {
	return o.matchAndWake(ctx, watchEventsTaskAttentionEvent(hookspkg.HookTaskRecovered, payload, o.now))
}

func (o *loopWatchEventsObserver) OnTaskRunTerminal(
	ctx context.Context,
	payload hookspkg.TaskRunLeasePayload,
) error {
	return errors.Join(
		o.matchAndWake(ctx, watchEventsTaskRunTerminalEvent(payload, o.now)),
		o.Hydrate(ctx),
	)
}

func (o *loopWatchEventsObserver) OnLoopTerminal(ctx context.Context, payload hookspkg.LoopTerminalPayload) error {
	return errors.Join(
		o.matchAndWake(ctx, watchEventsLoopTerminalEvent(payload, o.now)),
		o.Hydrate(ctx),
	)
}

func (o *loopWatchEventsObserver) OnLoopNodeTerminal(
	ctx context.Context,
	payload hookspkg.LoopNodeTerminalPayload,
) error {
	return o.matchAndWake(ctx, watchEventsLoopNodeTerminalEvent(payload, o.now))
}

func (o *loopWatchEventsObserver) OnAutomationRunCompleted(
	ctx context.Context,
	payload hookspkg.AutomationRunCompletedPayload,
) error {
	return o.matchAndWake(ctx, watchEventsAutomationRunCompletedEvent(payload, o.now))
}

func (o *loopWatchEventsObserver) OnAutomationRunFailed(
	ctx context.Context,
	payload hookspkg.AutomationRunFailedPayload,
) error {
	return o.matchAndWake(ctx, watchEventsAutomationRunFailedEvent(payload, o.now))
}

func (o *loopWatchEventsObserver) OnNetworkThreadOpened(
	ctx context.Context,
	payload hookspkg.NetworkThreadOpenedPayload,
) error {
	return o.matchAndWake(ctx, watchEventsNetworkEvent(hookspkg.HookNetworkThreadOpened, payload, o.now))
}

func (o *loopWatchEventsObserver) OnNetworkDirectRoomOpened(
	ctx context.Context,
	payload hookspkg.NetworkDirectRoomOpenedPayload,
) error {
	return o.matchAndWake(ctx, watchEventsNetworkEvent(hookspkg.HookNetworkDirectRoomOpened, payload, o.now))
}

func (o *loopWatchEventsObserver) OnNetworkMessagePersisted(
	ctx context.Context,
	payload hookspkg.NetworkMessagePersistedPayload,
) error {
	return o.matchAndWake(ctx, watchEventsNetworkEvent(hookspkg.HookNetworkMessagePersisted, payload, o.now))
}

func (o *loopWatchEventsObserver) OnNetworkWorkOpened(
	ctx context.Context,
	payload hookspkg.NetworkWorkOpenedPayload,
) error {
	return o.matchAndWake(ctx, watchEventsNetworkEvent(hookspkg.HookNetworkWorkOpened, payload, o.now))
}

func (o *loopWatchEventsObserver) OnNetworkWorkTransitioned(
	ctx context.Context,
	payload hookspkg.NetworkWorkTransitionedPayload,
) error {
	return o.matchAndWake(ctx, watchEventsNetworkEvent(hookspkg.HookNetworkWorkTransitioned, payload, o.now))
}

func (o *loopWatchEventsObserver) OnNetworkWorkClosed(
	ctx context.Context,
	payload hookspkg.NetworkWorkClosedPayload,
) error {
	return o.matchAndWake(ctx, watchEventsNetworkEvent(hookspkg.HookNetworkWorkClosed, payload, o.now))
}

func (o *loopWatchEventsObserver) matchAndWake(ctx context.Context, event looppkg.WatchEvent) error {
	subscriptions := o.subscriptionsForKind(hookspkg.HookEvent(strings.TrimSpace(event.Kind)))
	if len(subscriptions) == 0 {
		return nil
	}
	var errs []error
	for _, subscription := range subscriptions {
		if !watchEventsWorkspaceMayMatch(subscription.WorkspaceID, event.WorkspaceID) ||
			!o.subscriptionMatchesEvent(subscription, event) {
			continue
		}
		if err := o.writeObserverEvent(
			ctx,
			loopWatchEventsMatchedEvent,
			subscription,
			event,
			taskpkg.Run{},
			false,
			nil,
		); err != nil {
			errs = append(errs, err)
		}
		run, added, err := o.store.EnqueueLoopCoordinatorWake(
			ctx,
			subscription.LoopRunID,
			loopWatchEventsWakeKey(subscription.LoopRunID, subscription.NodeID),
			o.actor.Origin,
			o.now().UTC(),
		)
		if err := normalizeLoopWakeError(err); err != nil {
			if writeErr := o.writeObserverEvent(
				ctx,
				loopWatchEventsWakeErrorEvent,
				subscription,
				event,
				taskpkg.Run{},
				false,
				err,
			); writeErr != nil {
				err = errors.Join(err, writeErr)
			}
			errs = append(errs, err)
			continue
		}
		if err := o.writeObserverEvent(
			ctx,
			loopWatchEventsWakeEnqueuedEvent,
			subscription,
			event,
			run,
			added,
			nil,
		); err != nil {
			errs = append(errs, err)
		}
		if added {
			errs = append(errs, o.startLoopCoordinatorBackstop(ctx))
		}
	}
	return errors.Join(errs...)
}

func (o *loopWatchEventsObserver) subscriptionsForKind(
	kind hookspkg.HookEvent,
) []looppkg.ParkedWatchEventSubscription {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return append([]looppkg.ParkedWatchEventSubscription(nil), o.byKind[kind]...)
}

func (o *loopWatchEventsObserver) subscriptionMatchesEvent(
	subscription looppkg.ParkedWatchEventSubscription,
	event looppkg.WatchEvent,
) bool {
	for _, ref := range subscription.Subscriptions {
		if strings.TrimSpace(ref.Kind) != strings.TrimSpace(event.Kind) {
			continue
		}
		if o.compiler.Match(ref.Filter, subscription.Inputs, event) {
			return true
		}
	}
	return false
}

func (o *loopWatchEventsObserver) writeObserverEvent(
	ctx context.Context,
	eventType string,
	subscription looppkg.ParkedWatchEventSubscription,
	event looppkg.WatchEvent,
	run taskpkg.Run,
	added bool,
	cause error,
) error {
	content, err := json.Marshal(map[string]any{
		loopWatchEventsContentLoopRunIDField:   strings.TrimSpace(subscription.LoopRunID),
		loopWatchEventsContentLoopNameField:    strings.TrimSpace(subscription.LoopName),
		watchEventsPayloadNodeIDKey:            strings.TrimSpace(subscription.NodeID),
		loopWatchEventsContentWorkspaceIDField: strings.TrimSpace(subscription.WorkspaceID),
		loopWatchEventsContentHookEventField:   strings.TrimSpace(event.Kind),
		loopWatchEventsContentEventTaskIDField: strings.TrimSpace(event.TaskID),
		loopWatchEventsContentEventRunIDField:  strings.TrimSpace(event.RunID),
		loopWatchEventsContentWakeRunIDField:   strings.TrimSpace(run.ID),
		loopWatchEventsContentIdempotencyField: loopWatchEventsWakeKey(subscription.LoopRunID, subscription.NodeID),
		loopWatchEventsContentAddedField:       added,
		watchEventsPayloadErrorKey:             loopWatchEventsErrorString(cause),
	})
	if err != nil {
		return fmt.Errorf("daemon: marshal loop watch-events observer event: %w", err)
	}
	return o.store.WriteEventSummary(ctx, store.EventSummary{
		SessionID:   loopWatchEventsObserverSessionID,
		WorkspaceID: strings.TrimSpace(subscription.WorkspaceID),
		Type:        eventType,
		AgentName:   loopWatchEventsDaemonAgentName,
		Outcome:     loopWatchEventsOutcome(eventType),
		Content:     content,
		EventCorrelation: store.EventCorrelation{
			TaskID:               strings.TrimSpace(event.TaskID),
			RunID:                firstNonEmptyWatchEventsValue(run.ID, event.RunID),
			CoordinatorSessionID: strings.TrimSpace(event.SessionID),
			SchedulerReason:      loopWatchEventsDoorbellReason,
			HookEvent:            strings.TrimSpace(event.Kind),
			ActorKind:            string(taskpkg.ActorKindDaemon),
			ActorID:              loopWatchEventsObserverActorID,
		},
		Summary:   loopWatchEventsSummary(eventType, event, subscription),
		Timestamp: o.now().UTC(),
	})
}

func (o *loopWatchEventsObserver) startLoopCoordinatorBackstop(ctx context.Context) error {
	_, err := o.backstop.RunLoopCoordinatorBackstop(ctx, o.now().UTC(), o.actor)
	return err
}

func loopWatchEventsWakeKey(loopRunID string, nodeID string) string {
	return fmt.Sprintf(
		"loop.coordinator.watch_events.%s.%s",
		strings.TrimSpace(loopRunID),
		strings.TrimSpace(nodeID),
	)
}

func watchEventsWorkspaceMayMatch(subscriptionWorkspaceID string, eventWorkspaceID string) bool {
	subscriptionWorkspaceID = strings.TrimSpace(subscriptionWorkspaceID)
	eventWorkspaceID = strings.TrimSpace(eventWorkspaceID)
	return subscriptionWorkspaceID == "" || eventWorkspaceID == "" || subscriptionWorkspaceID == eventWorkspaceID
}

func loopWatchEventsOutcome(eventType string) string {
	if eventType == loopWatchEventsWakeErrorEvent {
		return string(eventspkg.OutcomeFailure)
	}
	return string(eventspkg.OutcomeInfo)
}

func loopWatchEventsSummary(
	eventType string,
	event looppkg.WatchEvent,
	subscription looppkg.ParkedWatchEventSubscription,
) string {
	switch eventType {
	case loopWatchEventsMatchedEvent:
		return "watch-events matched " + strings.TrimSpace(
			event.Kind,
		) + " for " + strings.TrimSpace(
			subscription.NodeID,
		)
	case loopWatchEventsWakeErrorEvent:
		return "watch-events wake failed for " + strings.TrimSpace(subscription.NodeID)
	default:
		return "watch-events wake enqueued for " + strings.TrimSpace(subscription.NodeID)
	}
}

func cloneParkedWatchEventSubscription(
	src looppkg.ParkedWatchEventSubscription,
) looppkg.ParkedWatchEventSubscription {
	src.Inputs = maps.Clone(src.Inputs)
	src.Subscriptions = append(src.Subscriptions[:0:0], src.Subscriptions...)
	src.Cursors = maps.Clone(src.Cursors)
	return src
}

func firstNonEmptyWatchEventsValue(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func loopWatchEventsErrorString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
