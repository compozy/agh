package sessiondb

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/compozy/agh/internal/store"
	"github.com/compozy/agh/internal/transcript"
)

type projectionDB interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

type projectionSQLResolver struct {
	db projectionDB
}

func (r projectionSQLResolver) EntryIdentity(
	ctx context.Context,
	key string,
) (transcript.EntryIdentity, bool, error) {
	return scanOptionalProjectionIdentity(r.db.QueryRowContext(ctx, `
		SELECT entry_key, kind, logical_id, turn_id, base_message_id,
		       COALESCE(message_id, ''), start_sequence, updated_sequence, complete
		FROM transcript_entries WHERE entry_key = ?`, key))
}

func (r projectionSQLResolver) ToolEntryIdentity(
	ctx context.Context,
	toolKey string,
) (transcript.EntryIdentity, bool, error) {
	return scanOptionalProjectionIdentity(r.db.QueryRowContext(ctx, `
		SELECT e.entry_key, e.kind, e.logical_id, e.turn_id, e.base_message_id,
		       COALESCE(e.message_id, ''), e.start_sequence, e.updated_sequence, e.complete
		FROM transcript_tool_routes AS r
		JOIN transcript_entries AS e ON e.entry_key = r.entry_key
		WHERE r.tool_key = ?`, toolKey))
}

func (r projectionSQLResolver) LatestAssistantIdentity(
	ctx context.Context,
	turnID string,
) (transcript.EntryIdentity, bool, error) {
	return scanOptionalProjectionIdentity(r.db.QueryRowContext(ctx, `
		SELECT entry_key, kind, logical_id, turn_id, base_message_id,
		       COALESCE(message_id, ''), start_sequence, updated_sequence, complete
		FROM transcript_entries
		WHERE kind = ? AND turn_id = ?
		ORDER BY start_sequence DESC LIMIT 1`, transcript.EntryKindAssistant, turnID))
}

func scanOptionalProjectionIdentity(
	row *sql.Row,
) (transcript.EntryIdentity, bool, error) {
	var identity transcript.EntryIdentity
	var kind string
	var complete int
	if err := row.Scan(
		&identity.Key,
		&kind,
		&identity.LogicalID,
		&identity.TurnID,
		&identity.BaseMessageID,
		&identity.MessageID,
		&identity.StartSequence,
		&identity.UpdatedSequence,
		&complete,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return transcript.EntryIdentity{}, false, nil
		}
		return transcript.EntryIdentity{}, false, fmt.Errorf("store: query transcript identity: %w", err)
	}
	identity.Kind = transcript.EntryKind(kind)
	identity.Complete = complete != 0
	return identity, true, nil
}

func loadProjectionState(ctx context.Context, db projectionDB) (transcript.ProjectionState, error) {
	var state transcript.ProjectionState
	if err := db.QueryRowContext(ctx, `
		SELECT projection_version, generation, COALESCE(active_entry_key, '')
		FROM transcript_projection_state WHERE singleton = 1`).Scan(
		&state.Version,
		&state.Generation,
		&state.ActiveEntryKey,
	); err != nil {
		return transcript.ProjectionState{}, fmt.Errorf("%w: load state: %v", transcript.ErrProjectionCorrupt, err)
	}
	if state.Version != transcript.ProjectionVersion {
		return transcript.ProjectionState{}, fmt.Errorf(
			"%w: stored version %d, supported version %d",
			transcript.ErrProjectionIncompatible,
			state.Version,
			transcript.ProjectionVersion,
		)
	}
	return state, nil
}

func persistProjectionState(ctx context.Context, db projectionDB, state transcript.ProjectionState) error {
	if _, err := db.ExecContext(ctx, `
		UPDATE transcript_projection_state
		SET projection_version = ?, generation = ?, active_entry_key = NULLIF(?, '')
		WHERE singleton = 1`, state.Version, state.Generation, state.ActiveEntryKey); err != nil {
		return fmt.Errorf("store: update transcript projection state: %w", err)
	}
	return nil
}

func persistProjectionEntry(
	ctx context.Context,
	db projectionDB,
	identity transcript.EntryIdentity,
	entry *transcript.Entry,
) error {
	var messageJSON any
	var eventType string
	var markerJSON any
	if entry != nil {
		encoded, err := json.Marshal(entry.Message)
		if err != nil {
			return fmt.Errorf("store: marshal transcript message: %w", err)
		}
		messageJSON = string(encoded)
		eventType = entry.EventType
		if entry.Marker != nil {
			encodedMarker, markerErr := json.Marshal(entry.Marker)
			if markerErr != nil {
				return fmt.Errorf("store: marshal transcript marker: %w", markerErr)
			}
			markerJSON = string(encodedMarker)
		}
	}
	if _, err := db.ExecContext(ctx, `
		INSERT INTO transcript_entries (
			entry_key, kind, logical_id, turn_id, base_message_id, message_id,
			start_sequence, updated_sequence, complete, message_json, event_type, marker_json
		) VALUES (?, ?, ?, ?, ?, NULLIF(?, ''), ?, ?, ?, ?, ?, ?)
		ON CONFLICT(entry_key) DO UPDATE SET
			logical_id = excluded.logical_id,
			turn_id = excluded.turn_id,
			base_message_id = excluded.base_message_id,
			message_id = excluded.message_id,
			updated_sequence = excluded.updated_sequence,
			complete = excluded.complete,
			message_json = excluded.message_json,
			event_type = excluded.event_type,
			marker_json = excluded.marker_json`,
		identity.Key,
		identity.Kind,
		identity.LogicalID,
		identity.TurnID,
		identity.BaseMessageID,
		identity.MessageID,
		identity.StartSequence,
		identity.UpdatedSequence,
		boolToSQLite(identity.Complete),
		messageJSON,
		eventType,
		markerJSON,
	); err != nil {
		return fmt.Errorf("store: upsert transcript entry %q: %w", identity.Key, err)
	}
	return nil
}

func allocateProjectionMessageID(
	ctx context.Context,
	db projectionDB,
	entryKey string,
	base string,
) (string, error) {
	base = strings.TrimSpace(base)
	if base == "" {
		base = "message"
	}
	for suffix := 1; ; suffix++ {
		candidate := base
		if suffix > 1 {
			candidate = fmt.Sprintf("%s-%d", base, suffix)
		}
		var count int
		if err := db.QueryRowContext(ctx, `
			SELECT COUNT(*) FROM transcript_entries
			WHERE message_id = ? AND entry_key <> ?`, candidate, entryKey).Scan(&count); err != nil {
			return "", fmt.Errorf("store: test transcript message id %q: %w", candidate, err)
		}
		if count == 0 {
			return candidate, nil
		}
	}
}

func loadAssignedEvents(
	ctx context.Context,
	db projectionDB,
	sessionID string,
	entryKey string,
) (events []store.SessionEvent, err error) {
	rows, err := db.QueryContext(ctx, `
		SELECT id, sequence, turn_id, type, agent_name, content, timestamp
		FROM events WHERE transcript_entry_key = ? ORDER BY sequence ASC`, entryKey)
	if err != nil {
		return nil, fmt.Errorf("store: query events for transcript entry %q: %w", entryKey, err)
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			err = errors.Join(err, fmt.Errorf("store: close events for transcript entry %q: %w", entryKey, closeErr))
		}
	}()
	events = make([]store.SessionEvent, 0)
	for rows.Next() {
		event, scanErr := scanProjectionEvent(rows, sessionID)
		if scanErr != nil {
			return nil, scanErr
		}
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: iterate events for transcript entry %q: %w", entryKey, err)
	}
	return events, nil
}

func scanProjectionEvent(scanner rowScanner, sessionID string) (store.SessionEvent, error) {
	var event store.SessionEvent
	var timestamp string
	if err := scanner.Scan(
		&event.ID,
		&event.Sequence,
		&event.TurnID,
		&event.Type,
		&event.AgentName,
		&event.Content,
		&timestamp,
	); err != nil {
		return store.SessionEvent{}, fmt.Errorf("store: scan transcript projection event: %w", err)
	}
	parsed, err := store.ParseTimestamp(timestamp)
	if err != nil {
		parsed, err = time.Parse(time.RFC3339Nano, timestamp)
		if err != nil {
			return store.SessionEvent{}, fmt.Errorf("store: parse transcript event timestamp %q: %w", timestamp, err)
		}
	}
	event.SessionID = sessionID
	event.Timestamp = parsed
	return event, nil
}
