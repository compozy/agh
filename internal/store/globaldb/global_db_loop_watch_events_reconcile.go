package globaldb

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	eventspkg "github.com/compozy/agh/internal/events"
	"github.com/compozy/agh/internal/hooks"
	looppkg "github.com/compozy/agh/internal/loop"
	"github.com/compozy/agh/internal/store"
	taskpkg "github.com/compozy/agh/internal/task"
)

const (
	watchEventsMatchedEvent      = "loop.watch_events.matched"
	watchEventsWakeEnqueuedEvent = "loop.watch_events.wake_enqueued"
	watchEventsWakeErrorEvent    = "loop.watch_events.wake_error"

	watchEventsRecoverySessionID = "loop-watch-events-recovery"
	watchEventsDaemonAgentName   = "daemon"
	watchEventsGapReason         = "watch_events_gap"

	watchEventsContentWorkspaceIDKey = "workspace_id"
	watchEventsContentIdempotencyKey = "idempotency_key"
	watchEventsContentRunIDKey       = "run_id"
	watchEventsContentErrorKey       = "error"
)

// EnqueueWatchEventsGapWakes scans parked watch-events nodes for durable
// cursor gaps and enqueues idempotent coordinator wakes.
func (g *GlobalDB) EnqueueWatchEventsGapWakes(
	ctx context.Context,
	origin taskpkg.Origin,
	now time.Time,
) ([]taskpkg.Run, error) {
	if err := g.checkReady(ctx, "enqueue watch-events gap wakes"); err != nil {
		return nil, err
	}
	normalizedOrigin, err := normalizeLoopCoordinatorReconcileOrigin(origin)
	if err != nil {
		return nil, err
	}
	now = g.normalizeLoopCoordinatorReconcileTime(now)

	parked, err := g.ListParkedWatchEventSubscriptions(ctx)
	if err != nil {
		return nil, err
	}
	enqueued := make([]taskpkg.Run, 0)
	for _, subscription := range parked {
		hasGap, err := g.parkedWatchEventSubscriptionHasGap(ctx, subscription)
		if err != nil {
			return nil, err
		}
		if !hasGap {
			continue
		}
		if err := g.writeWatchEventsGapEvent(
			ctx,
			subscription,
			taskpkg.Run{},
			watchEventsMatchedEvent,
			string(eventspkg.OutcomeInfo),
			now,
			nil,
		); err != nil {
			return nil, err
		}
		run, added, err := g.EnqueueLoopCoordinatorWake(
			ctx,
			subscription.LoopRunID,
			watchEventsCoordinatorWakeKey(subscription.LoopRunID, subscription.NodeID),
			normalizedOrigin,
			now,
		)
		if err := normalizeWatchEventsGapWakeError(err); err != nil {
			if writeErr := g.writeWatchEventsGapEvent(
				ctx,
				subscription,
				taskpkg.Run{},
				watchEventsWakeErrorEvent,
				string(eventspkg.OutcomeFailure),
				now,
				err,
			); writeErr != nil {
				return nil, errors.Join(err, writeErr)
			}
			return nil, err
		}
		if err := g.writeWatchEventsGapEvent(
			ctx,
			subscription,
			run,
			watchEventsWakeEnqueuedEvent,
			string(eventspkg.OutcomeInfo),
			now,
			nil,
		); err != nil {
			return nil, err
		}
		if added {
			enqueued = append(enqueued, run)
		}
	}
	return enqueued, nil
}

func (g *GlobalDB) parkedWatchEventSubscriptionHasGap(
	ctx context.Context,
	subscription looppkg.ParkedWatchEventSubscription,
) (bool, error) {
	query, err := watchEventsGapQuery(subscription)
	if err != nil {
		return false, err
	}
	cursors, err := g.ReadCursors(ctx, query)
	if err != nil {
		return false, err
	}
	for stream, cursor := range cursors {
		if cursor > subscription.Cursors[stream] {
			return true, nil
		}
	}
	return false, nil
}

func watchEventsGapQuery(
	subscription looppkg.ParkedWatchEventSubscription,
) (looppkg.WatchEventsQuery, error) {
	streams := map[string]int64{}
	kindSet := map[string]struct{}{}
	supported := looppkg.SupportedWatchEvents()
	for _, ref := range subscription.Subscriptions {
		contract, ok := supported[hooks.HookEvent(strings.TrimSpace(ref.Kind))]
		if !ok {
			return looppkg.WatchEventsQuery{}, fmt.Errorf(
				"%w: watch-events kind is unsupported: %q",
				looppkg.ErrValidation,
				ref.Kind,
			)
		}
		matchedCursor := false
		for stream, cursor := range subscription.Cursors {
			if looppkg.WatchEventsBaseStream(stream) != contract.Stream {
				continue
			}
			streams[stream] = cursor
			matchedCursor = true
		}
		if !matchedCursor {
			return looppkg.WatchEventsQuery{}, fmt.Errorf(
				"%w: watch-events cursor for stream %q is required",
				looppkg.ErrValidation,
				contract.Stream,
			)
		}
		for _, ledgerType := range contract.LedgerTypes {
			trimmed := strings.TrimSpace(ledgerType)
			if trimmed != "" {
				kindSet[trimmed] = struct{}{}
			}
		}
	}
	kinds := make([]string, 0, len(kindSet))
	for kind := range kindSet {
		kinds = append(kinds, kind)
	}
	return looppkg.WatchEventsQuery{
		WorkspaceID: strings.TrimSpace(subscription.WorkspaceID),
		Streams:     streams,
		Kinds:       kinds,
		Limit:       looppkg.LoopMaxFanoutWidth,
	}, nil
}

func (g *GlobalDB) writeWatchEventsGapEvent(
	ctx context.Context,
	subscription looppkg.ParkedWatchEventSubscription,
	run taskpkg.Run,
	eventType string,
	outcome string,
	now time.Time,
	cause error,
) error {
	content, err := json.Marshal(map[string]any{
		columnLoopRunID:                  strings.TrimSpace(subscription.LoopRunID),
		columnLoopName:                   strings.TrimSpace(subscription.LoopName),
		loopRunEventPayloadKeyNodeID:     strings.TrimSpace(subscription.NodeID),
		watchEventsContentWorkspaceIDKey: strings.TrimSpace(subscription.WorkspaceID),
		loopRunEventPayloadKeyGeneration: subscription.Generation,
		watchEventsContentIdempotencyKey: watchEventsCoordinatorWakeKey(subscription.LoopRunID, subscription.NodeID),
		watchEventsContentRunIDKey:       strings.TrimSpace(run.ID),
		watchEventsContentErrorKey:       watchEventsErrorString(cause),
	})
	if err != nil {
		return fmt.Errorf("store: marshal watch-events gap event: %w", err)
	}
	return g.WriteEventSummary(ctx, store.EventSummary{
		SessionID:   watchEventsRecoverySessionID,
		WorkspaceID: strings.TrimSpace(subscription.WorkspaceID),
		Type:        eventType,
		AgentName:   watchEventsDaemonAgentName,
		Outcome:     outcome,
		Content:     content,
		EventCorrelation: store.EventCorrelation{
			RunID:           strings.TrimSpace(run.ID),
			SchedulerReason: watchEventsGapReason,
			ActorKind:       string(taskpkg.ActorKindDaemon),
			ActorID:         "loop-watch-events-recovery",
		},
		Summary:   watchEventsGapSummary(eventType, subscription),
		Timestamp: now.UTC(),
	})
}

func watchEventsGapSummary(eventType string, subscription looppkg.ParkedWatchEventSubscription) string {
	switch eventType {
	case watchEventsMatchedEvent:
		return "watch-events gap matched for " + strings.TrimSpace(subscription.NodeID)
	case watchEventsWakeErrorEvent:
		return "watch-events gap wake failed for " + strings.TrimSpace(subscription.NodeID)
	default:
		return "watch-events gap wake enqueued for " + strings.TrimSpace(subscription.NodeID)
	}
}

func watchEventsCoordinatorWakeKey(loopRunID string, nodeID string) string {
	return fmt.Sprintf(
		"loop.coordinator.watch_events.%s.%s",
		strings.TrimSpace(loopRunID),
		strings.TrimSpace(nodeID),
	)
}

func normalizeWatchEventsGapWakeError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, taskpkg.ErrInvalidStatusTransition), errors.Is(err, taskpkg.ErrConflict):
		return nil
	default:
		return err
	}
}

func watchEventsErrorString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
