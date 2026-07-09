package task

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	hookspkg "github.com/compozy/agh/internal/hooks"
)

const (
	taskWatchEventBlocked        = string(hookspkg.HookTaskBlocked)
	taskWatchEventUnblocked      = string(hookspkg.HookTaskUnblocked)
	taskWatchEventNeedsAttention = string(hookspkg.HookTaskNeedsAttention)
	taskWatchEventRecovered      = string(hookspkg.HookTaskRecovered)
	taskWatchEventStatusChanged  = string(hookspkg.HookTaskStatusChanged)
	taskWatchEventRunCompleted   = string(hookspkg.HookTaskRunCompleted)
	taskWatchEventRunFailed      = string(hookspkg.HookTaskRunFailed)
)

func (m *Service) recordTaskEvent(
	ctx context.Context,
	taskID string,
	runID string,
	eventType string,
	actor ActorContext,
	payload any,
) error {
	if isTransactionalWatchTaskEvent(eventType) {
		return fmt.Errorf(
			"%w: task event %q must be appended inside its owning transaction",
			ErrValidation,
			strings.TrimSpace(eventType),
		)
	}
	rawPayload, err := marshalTaskEventPayload(payload)
	if err != nil {
		return err
	}
	event := Event{
		ID:        m.newID("evt"),
		TaskID:    strings.TrimSpace(taskID),
		RunID:     strings.TrimSpace(runID),
		EventType: strings.TrimSpace(eventType),
		Actor:     actor.Actor,
		Origin:    actor.Origin,
		Payload:   rawPayload,
		Timestamp: m.now().UTC(),
	}
	if err := m.store.CreateTaskEvent(ctx, event); err != nil {
		return err
	}

	postCommitCtx := context.Background()
	if ctx != nil {
		postCommitCtx = context.WithoutCancel(ctx)
	}
	record, err := m.store.GetTaskEventRecord(postCommitCtx, event.ID)
	if err != nil {
		m.emitTaskLiveEventBestEffort(postCommitCtx, event.ID)
		return nil
	}
	m.notifyTaskObserverBestEffort(postCommitCtx, record)
	m.emitTaskLiveRecordBestEffort(postCommitCtx, record)
	return nil
}

func isTransactionalWatchTaskEvent(eventType string) bool {
	switch strings.TrimSpace(eventType) {
	case taskWatchEventBlocked,
		taskWatchEventUnblocked,
		taskWatchEventNeedsAttention,
		taskWatchEventRecovered,
		taskWatchEventStatusChanged,
		taskWatchEventRunCompleted,
		taskWatchEventRunFailed:
		return true
	default:
		return false
	}
}

func (m *Service) notifyTaskObserverBestEffort(ctx context.Context, record EventRecord) {
	if m == nil || m.eventObserver == nil {
		return
	}
	defer func() {
		if recovered := recover(); recovered != nil {
			slog.Error(
				"task: task event observer panicked during post-commit notification",
				"panic", recovered,
				"event_id", record.Event.ID,
				"task_id", record.Event.TaskID,
				"run_id", record.Event.RunID,
				"event_type", record.Event.EventType,
			)
		}
	}()
	m.eventObserver.OnTaskEvent(ctx, record)
}

func marshalTaskEventPayload(payload any) (json.RawMessage, error) {
	if payload == nil {
		return nil, nil
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("task: marshal task event payload: %w", err)
	}
	return json.RawMessage(raw), nil
}
