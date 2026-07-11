package sessiondb

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/compozy/agh/internal/store"
	"github.com/compozy/agh/internal/transcript"
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
		state, err := loadProjectionState(ctx, tx)
		if err != nil {
			return err
		}
		projector, err := transcript.NewProjector(state, projectionSQLResolver{db: tx})
		if err != nil {
			return err
		}
		nextSequence, err := nextEventSequence(ctx, tx)
		if err != nil {
			return err
		}
		affected := make(map[string]struct{})
		for idx := range persisted {
			persisted[idx].Sequence = nextSequence + int64(idx)
			assignment, assignErr := projector.Assign(ctx, persisted[idx])
			if assignErr != nil {
				return assignErr
			}
			if err := insertSessionEvent(ctx, tx, persisted[idx], assignment.Entry.Key); err != nil {
				return err
			}
			affected[assignment.Entry.Key] = struct{}{}
			for _, key := range assignment.CompletedKeys {
				affected[key] = struct{}{}
			}
		}
		return persistIncrementalTranscriptProjection(ctx, tx, s.sessionID, projector, affected)
	}); err != nil {
		return nil, err
	}

	return persisted, nil
}

func insertSessionEvent(
	ctx context.Context,
	tx *store.WriteTx,
	event store.SessionEvent,
	entryKey string,
) error {
	if _, err := tx.ExecContext(
		ctx,
		`INSERT INTO events (
			id, sequence, turn_id, type, agent_name, content, timestamp, transcript_entry_key
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		event.ID,
		event.Sequence,
		event.TurnID,
		event.Type,
		event.AgentName,
		event.Content,
		store.FormatTimestamp(event.Timestamp),
		entryKey,
	); err != nil {
		return fmt.Errorf("store: insert session event: %w", err)
	}
	return nil
}

func persistIncrementalTranscriptProjection(
	ctx context.Context,
	tx *store.WriteTx,
	sessionID string,
	projector *transcript.Projector,
	affected map[string]struct{},
) error {
	identities := make([]transcript.EntryIdentity, 0, len(affected))
	for key := range affected {
		identity, ok := projector.Identity(key)
		if !ok {
			return fmt.Errorf("%w: missing affected identity %q", transcript.ErrProjectionCorrupt, key)
		}
		identities = append(identities, identity)
	}
	sort.Slice(identities, func(i, j int) bool {
		return identities[i].StartSequence < identities[j].StartSequence
	})

	// Each append rewrites only the independently rebuildable entries it affects.
	// This bounded write amplification is required to preserve full-message
	// semantics while reads stay proportional to the requested result window.
	for _, identity := range identities {
		events, err := loadAssignedEvents(ctx, tx, sessionID, identity.Key)
		if err != nil {
			return err
		}
		candidate := identity
		if candidate.MessageID == "" {
			candidate.MessageID = candidate.BaseMessageID
		}
		entry, err := transcript.ProjectAssignedEntry(events, candidate)
		if err != nil {
			return fmt.Errorf("store: project transcript entry %q: %w", identity.Key, err)
		}
		if entry != nil && identity.MessageID == "" {
			identity.MessageID, err = allocateProjectionMessageID(
				ctx,
				tx,
				identity.Key,
				identity.BaseMessageID,
			)
			if err != nil {
				return err
			}
			entry, err = transcript.ProjectAssignedEntry(events, identity)
			if err != nil {
				return fmt.Errorf("store: reproject transcript entry %q: %w", identity.Key, err)
			}
		}
		if err := persistProjectionEntry(ctx, tx, identity, entry); err != nil {
			return err
		}
	}
	for toolKey, entryKey := range projector.ToolRoutes() {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO transcript_tool_routes (tool_key, entry_key) VALUES (?, ?)
			ON CONFLICT(tool_key) DO UPDATE SET entry_key = excluded.entry_key`,
			toolKey,
			entryKey,
		); err != nil {
			return fmt.Errorf("store: upsert transcript tool route %q: %w", toolKey, err)
		}
	}
	return persistProjectionState(ctx, tx, projector.State())
}
