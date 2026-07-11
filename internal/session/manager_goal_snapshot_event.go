package session

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/compozy/agh/internal/store"
)

type idempotentSessionEventAppender interface {
	AppendEventIfAbsent(context.Context, store.SessionEvent) (store.SessionEvent, error)
}

// AppendSessionEventIfAbsent persists and publishes one deterministic Goal projection event.
func (m *Manager) AppendSessionEventIfAbsent(ctx context.Context, event GoalEvent) error {
	if m == nil {
		return errors.New("session: manager is required")
	}
	if ctx == nil {
		return errors.New("session: append Goal event context is required")
	}
	if err := event.Validate(); err != nil {
		return err
	}
	target := strings.TrimSpace(event.SessionID)
	if _, err := m.Status(ctx, target); err != nil {
		return err
	}
	persisted, err := m.appendGoalSnapshotEvent(ctx, target, store.SessionEvent{
		ID:        strings.TrimSpace(event.EventID),
		SessionID: target,
		TurnID:    strings.TrimSpace(event.SyntheticTurnID),
		Type:      EventTypeGoalSnapshotChanged,
		AgentName: strings.TrimSpace(event.AgentName),
		Content:   string(event.Content),
		Timestamp: event.CreatedAt.UTC(),
	})
	if err != nil {
		return err
	}
	m.publishSessionEventByID(ctx, target, persisted)
	return nil
}

func (m *Manager) appendGoalSnapshotEvent(
	ctx context.Context,
	sessionID string,
	event store.SessionEvent,
) (store.SessionEvent, error) {
	for attempt := range 2 {
		persisted, err := m.appendGoalSnapshotEventAttempt(ctx, sessionID, event)
		if err == nil || !errors.Is(err, store.ErrClosed) || attempt == 1 {
			return persisted, err
		}
		if _, waitErr := m.waitForSessionFinalization(ctx, sessionID); waitErr != nil {
			return store.SessionEvent{}, fmt.Errorf(
				"session: wait for finalization for %q after closed Goal event recorder: %w",
				sessionID,
				waitErr,
			)
		}
	}
	return store.SessionEvent{}, store.ErrClosed
}

func (m *Manager) appendGoalSnapshotEventAttempt(
	ctx context.Context,
	sessionID string,
	event store.SessionEvent,
) (store.SessionEvent, error) {
	if active, ok := m.Get(sessionID); ok {
		return appendGoalSnapshotEventWithRecorder(ctx, active.recorderHandle(), event)
	}
	if _, err := m.waitForSessionFinalization(ctx, sessionID); err != nil {
		return store.SessionEvent{}, err
	}
	path := store.SessionDBFile(filepath.Join(m.homePaths.SessionsDir, sessionID))
	recorder, err := m.openStore(ctx, sessionID, path)
	if err != nil {
		return store.SessionEvent{}, err
	}
	persisted, appendErr := appendGoalSnapshotEventWithRecorder(ctx, recorder, event)
	closeErr := recorder.Close(ctx)
	if appendErr != nil || closeErr != nil {
		return store.SessionEvent{}, errors.Join(appendErr, closeErr)
	}
	return persisted, nil
}

func appendGoalSnapshotEventWithRecorder(
	ctx context.Context,
	recorder EventRecorder,
	event store.SessionEvent,
) (store.SessionEvent, error) {
	appender, ok := recorder.(idempotentSessionEventAppender)
	if !ok {
		return store.SessionEvent{}, fmt.Errorf(
			"session: event recorder %T does not support idempotent append",
			recorder,
		)
	}
	return appender.AppendEventIfAbsent(ctx, event)
}
