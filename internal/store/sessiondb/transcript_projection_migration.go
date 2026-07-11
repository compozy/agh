package sessiondb

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"slices"

	"github.com/compozy/agh/internal/store"
	"github.com/compozy/agh/internal/transcript"
)

var transcriptProjectionSchemaStatements = []string{
	`ALTER TABLE events ADD COLUMN transcript_entry_key TEXT NOT NULL DEFAULT '';`,
	`CREATE INDEX idx_events_transcript_entry_sequence
		ON events(transcript_entry_key, sequence);`,
	`CREATE TABLE transcript_projection_state (
		singleton INTEGER PRIMARY KEY CHECK(singleton = 1),
		projection_version INTEGER NOT NULL,
		generation INTEGER NOT NULL,
		active_entry_key TEXT
	);`,
	`CREATE TABLE transcript_entries (
		entry_key TEXT PRIMARY KEY,
		kind TEXT NOT NULL,
		logical_id TEXT NOT NULL,
		turn_id TEXT NOT NULL,
		base_message_id TEXT NOT NULL,
		message_id TEXT,
		start_sequence INTEGER NOT NULL UNIQUE,
		updated_sequence INTEGER NOT NULL,
		complete INTEGER NOT NULL,
		message_json TEXT,
		event_type TEXT NOT NULL DEFAULT '',
		marker_json TEXT,
		CHECK(updated_sequence >= start_sequence)
	);`,
	`CREATE UNIQUE INDEX idx_transcript_entries_message_id
		ON transcript_entries(message_id) WHERE message_id IS NOT NULL;`,
	`CREATE INDEX idx_transcript_entries_visible_start
		ON transcript_entries(start_sequence) WHERE message_json IS NOT NULL;`,
	`CREATE INDEX idx_transcript_entries_updated
		ON transcript_entries(updated_sequence, start_sequence);`,
	`CREATE INDEX idx_transcript_entries_turn
		ON transcript_entries(kind, turn_id, start_sequence);`,
	`CREATE TABLE transcript_tool_routes (
		tool_key TEXT PRIMARY KEY,
		entry_key TEXT NOT NULL
	);`,
	`INSERT INTO transcript_projection_state (
		singleton, projection_version, generation, active_entry_key
	) VALUES (1, 1, 0, NULL);`,
}

func materializeTranscriptProjection(ctx context.Context, tx *sql.Tx) error {
	for _, statement := range transcriptProjectionSchemaStatements {
		if _, err := tx.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("store: create transcript projection schema: %w", err)
		}
	}
	events, err := loadMigrationEvents(ctx, tx)
	if err != nil {
		return err
	}
	projection, err := transcript.BuildProjection(events, 0)
	if err != nil {
		return fmt.Errorf("store: project legacy transcript: %w", err)
	}
	for eventID, entryKey := range projection.EventEntryKeys {
		if _, err := tx.ExecContext(
			ctx,
			`UPDATE events SET transcript_entry_key = ? WHERE id = ?`,
			entryKey,
			eventID,
		); err != nil {
			return fmt.Errorf("store: assign legacy transcript event %q: %w", eventID, err)
		}
	}
	for _, segment := range projection.Segments {
		if err := persistProjectionEntry(ctx, tx, segment.Identity, segment.Entry); err != nil {
			return err
		}
	}
	for toolKey, entryKey := range projection.ToolRoutes {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO transcript_tool_routes (tool_key, entry_key) VALUES (?, ?)`,
			toolKey,
			entryKey,
		); err != nil {
			return fmt.Errorf("store: backfill transcript tool route %q: %w", toolKey, err)
		}
	}
	if err := persistProjectionState(ctx, tx, projection.State); err != nil {
		return err
	}
	if err := verifyTranscriptProjectionMigration(ctx, tx, projection); err != nil {
		return err
	}
	return nil
}

func loadMigrationEvents(ctx context.Context, tx *sql.Tx) (events []store.SessionEvent, err error) {
	rows, err := tx.QueryContext(ctx, `
		SELECT id, sequence, turn_id, type, agent_name, content, timestamp
		FROM events ORDER BY sequence ASC`)
	if err != nil {
		return nil, fmt.Errorf("store: query legacy transcript events: %w", err)
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			err = errors.Join(err, fmt.Errorf("store: close legacy transcript events: %w", closeErr))
		}
	}()
	events = make([]store.SessionEvent, 0)
	for rows.Next() {
		event, scanErr := scanProjectionEvent(rows, "")
		if scanErr != nil {
			return nil, scanErr
		}
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: iterate legacy transcript events: %w", err)
	}
	return events, nil
}

func verifyTranscriptProjectionMigration(
	ctx context.Context,
	tx *sql.Tx,
	projection transcript.Projection,
) error {
	want := make([]transcript.Entry, 0, len(projection.Segments))
	for _, segment := range projection.Segments {
		if segment.Entry != nil {
			want = append(want, *segment.Entry)
		}
	}
	got, err := walkMigrationProjectionPages(ctx, tx, 200)
	if err != nil {
		return err
	}
	wantJSON, err := json.Marshal(want)
	if err != nil {
		return fmt.Errorf("store: marshal expected transcript migration projection: %w", err)
	}
	gotJSON, err := json.Marshal(got)
	if err != nil {
		return fmt.Errorf("store: marshal stored transcript migration projection: %w", err)
	}
	if !bytes.Equal(gotJSON, wantJSON) {
		return fmt.Errorf("%w: migration replay mismatch", transcript.ErrProjectionCorrupt)
	}
	return nil
}

func walkMigrationProjectionPages(
	ctx context.Context,
	tx *sql.Tx,
	limit int,
) ([]transcript.Entry, error) {
	entries := make([]transcript.Entry, 0)
	var before int64
	for {
		rows, err := queryMigrationProjectionPage(ctx, tx, before, limit+1)
		if err != nil {
			return nil, fmt.Errorf("store: verify transcript projection page: %w", err)
		}
		page, err := scanMaterializedEntries(rows)
		if err != nil {
			return nil, err
		}
		hasOlder := len(page) > limit
		if hasOlder {
			page = page[:limit]
		}
		slices.Reverse(page)
		entries = append(page, entries...)
		if !hasOlder {
			return entries, nil
		}
		before = page[0].StartSequence
	}
}

func queryMigrationProjectionPage(
	ctx context.Context,
	tx *sql.Tx,
	before int64,
	limit int,
) (*sql.Rows, error) {
	if before == 0 {
		return tx.QueryContext(ctx, `
			SELECT message_json, start_sequence, updated_sequence, event_type, marker_json
			FROM transcript_entries
			WHERE message_json IS NOT NULL
			ORDER BY start_sequence DESC LIMIT ?`, limit)
	}
	return tx.QueryContext(ctx, `
		SELECT message_json, start_sequence, updated_sequence, event_type, marker_json
		FROM transcript_entries
		WHERE message_json IS NOT NULL AND start_sequence < ?
		ORDER BY start_sequence DESC LIMIT ?`, before, limit)
}
