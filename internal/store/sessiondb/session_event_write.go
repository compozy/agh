package sessiondb

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/compozy/agh/internal/store"
)

// Record appends a session event using the dedicated writer goroutine.
func (s *SessionDB) Record(ctx context.Context, event store.SessionEvent) error {
	_, err := s.recordSessionEvent(ctx, event)
	return err
}

// RecordPersisted appends a session event and returns the stored row with sequence metadata.
func (s *SessionDB) RecordPersisted(ctx context.Context, event store.SessionEvent) (store.SessionEvent, error) {
	return s.recordSessionEvent(ctx, event)
}

// RecordPersistedBatch appends session events in one writer-owned transaction.
func (s *SessionDB) RecordPersistedBatch(
	ctx context.Context,
	events []store.SessionEvent,
) ([]store.SessionEvent, error) {
	if s == nil {
		return nil, errors.New("store: session database is required")
	}
	if ctx == nil {
		return nil, errors.New("store: record event batch context is required")
	}
	if len(events) == 0 {
		return nil, nil
	}
	normalized := make([]store.SessionEvent, 0, len(events))
	for _, event := range events {
		prepared, err := s.normalizeSessionEvent(event)
		if err != nil {
			return nil, err
		}
		normalized = append(normalized, prepared)
	}
	return s.enqueueWritePersistedBatch(ctx, sessionWriteRequest{
		ctx:    ctx,
		kind:   sessionWriteEventBatch,
		events: normalized,
		result: make(chan sessionWriteResult, 1),
	})
}

func (s *SessionDB) recordSessionEvent(
	ctx context.Context,
	event store.SessionEvent,
) (store.SessionEvent, error) {
	if s == nil {
		return store.SessionEvent{}, errors.New("store: session database is required")
	}
	if ctx == nil {
		return store.SessionEvent{}, errors.New("store: record event context is required")
	}
	normalized, err := s.normalizeSessionEvent(event)
	if err != nil {
		return store.SessionEvent{}, err
	}
	return s.enqueueWritePersisted(ctx, sessionWriteRequest{
		ctx:    ctx,
		kind:   sessionWriteEvent,
		event:  normalized,
		result: make(chan sessionWriteResult, 1),
	})
}

func (s *SessionDB) normalizeSessionEvent(event store.SessionEvent) (store.SessionEvent, error) {
	if err := event.Validate(); err != nil {
		return store.SessionEvent{}, err
	}
	if event.SessionID != "" && event.SessionID != s.sessionID {
		return store.SessionEvent{}, fmt.Errorf(
			"store: event session id %q does not match session database %q",
			event.SessionID,
			s.sessionID,
		)
	}
	event.SessionID = s.sessionID
	return event, nil
}

func (s *SessionDB) enqueueWritePersisted(
	ctx context.Context,
	req sessionWriteRequest,
) (store.SessionEvent, error) {
	s.acceptMu.RLock()
	defer s.acceptMu.RUnlock()

	if s.state.Load() != sessionStateOpen {
		return store.SessionEvent{}, store.ErrClosed
	}

	select {
	case s.writeCh <- req:
	case <-ctx.Done():
		return store.SessionEvent{}, fmt.Errorf("store: enqueue session write: %w", ctx.Err())
	}

	select {
	case result := <-req.result:
		if result.err != nil {
			return store.SessionEvent{}, result.err
		}
		return result.event, nil
	case <-ctx.Done():
		return store.SessionEvent{}, fmt.Errorf("store: wait for session write completion: %w", ctx.Err())
	}
}

func (s *SessionDB) enqueueWritePersistedBatch(
	ctx context.Context,
	req sessionWriteRequest,
) ([]store.SessionEvent, error) {
	s.acceptMu.RLock()
	defer s.acceptMu.RUnlock()

	if s.state.Load() != sessionStateOpen {
		return nil, store.ErrClosed
	}

	select {
	case s.writeCh <- req:
	case <-ctx.Done():
		return nil, fmt.Errorf("store: enqueue session write batch: %w", ctx.Err())
	}

	select {
	case result := <-req.result:
		if result.err != nil {
			return nil, result.err
		}
		return result.events, nil
	case <-ctx.Done():
		return nil, fmt.Errorf("store: wait for session write batch completion: %w", ctx.Err())
	}
}

func (s *SessionDB) writeEvent(ctx context.Context, event store.SessionEvent) (store.SessionEvent, error) {
	persisted, err := s.writeEventBatch(ctx, []store.SessionEvent{event})
	if err != nil {
		return store.SessionEvent{}, err
	}
	if len(persisted) != 1 {
		return store.SessionEvent{}, fmt.Errorf("store: persisted %d rows for one session event", len(persisted))
	}
	return persisted[0], nil
}

func (s *SessionDB) writeEventBatch(
	ctx context.Context,
	events []store.SessionEvent,
) ([]store.SessionEvent, error) {
	if len(events) == 0 {
		return nil, nil
	}
	prepared := make([]store.SessionEvent, 0, len(events))
	for _, event := range events {
		if strings.TrimSpace(event.ID) == "" {
			event.ID = store.NewID("ev")
		}
		if event.Timestamp.IsZero() {
			event.Timestamp = s.now()
		}
		prepared = append(prepared, event)
	}
	persisted, err := coalesceSessionEventBatch(prepared)
	if err != nil {
		return nil, err
	}

	if err := store.ExecuteWriteNoCheckpoint(ctx, s.db, func(ctx context.Context, tx *store.WriteTx) error {
		nextSequence, err := nextEventSequence(ctx, tx)
		if err != nil {
			return err
		}
		for idx := range persisted {
			persisted[idx].Sequence = nextSequence + int64(idx)
			if err := insertSessionEvent(ctx, tx, persisted[idx]); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return nil, err
	}

	return persisted, nil
}

func insertSessionEvent(ctx context.Context, tx *store.WriteTx, event store.SessionEvent) error {
	if _, err := tx.ExecContext(
		ctx,
		`INSERT INTO events (id, sequence, turn_id, type, agent_name, content, timestamp)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		event.ID,
		event.Sequence,
		event.TurnID,
		event.Type,
		event.AgentName,
		event.Content,
		store.FormatTimestamp(event.Timestamp),
	); err != nil {
		return fmt.Errorf("store: insert session event: %w", err)
	}
	return nil
}
