package sessiondb

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"slices"

	"github.com/compozy/agh/internal/store"
	"github.com/compozy/agh/internal/transcript"
)

var (
	_ transcript.Reader = (*SessionDB)(nil)
	_ transcript.Reader = (*ReadOnlySessionDB)(nil)
)

// TranscriptPage returns one bounded materialized transcript window.
func (s *SessionDB) TranscriptPage(
	ctx context.Context,
	query transcript.PageQuery,
) (transcript.Page, error) {
	if s == nil || s.db == nil {
		return transcript.Page{}, errors.New("store: session database is required")
	}
	if ctx == nil {
		return transcript.Page{}, errors.New("store: transcript page context is required")
	}
	s.acceptMu.RLock()
	defer s.acceptMu.RUnlock()
	if s.state.Load() != sessionStateOpen {
		return transcript.Page{}, store.ErrClosed
	}
	return queryTranscriptPage(ctx, s.db, query)
}

// TranscriptChanges returns entries changed by events after the supplied cursor.
func (s *SessionDB) TranscriptChanges(
	ctx context.Context,
	query transcript.ChangeQuery,
) (transcript.ChangePage, error) {
	if s == nil || s.db == nil {
		return transcript.ChangePage{}, errors.New("store: session database is required")
	}
	if ctx == nil {
		return transcript.ChangePage{}, errors.New("store: transcript changes context is required")
	}
	s.acceptMu.RLock()
	defer s.acceptMu.RUnlock()
	if s.state.Load() != sessionStateOpen {
		return transcript.ChangePage{}, store.ErrClosed
	}
	return queryTranscriptChanges(ctx, s.db, query)
}

// TranscriptPage returns one bounded materialized transcript window.
func (s *ReadOnlySessionDB) TranscriptPage(
	ctx context.Context,
	query transcript.PageQuery,
) (transcript.Page, error) {
	if s == nil || s.db == nil {
		return transcript.Page{}, errors.New("store: read-only session database is required")
	}
	if ctx == nil {
		return transcript.Page{}, errors.New("store: read-only transcript page context is required")
	}
	return queryTranscriptPage(ctx, s.db, query)
}

// TranscriptChanges returns entries changed by events after the supplied cursor.
func (s *ReadOnlySessionDB) TranscriptChanges(
	ctx context.Context,
	query transcript.ChangeQuery,
) (transcript.ChangePage, error) {
	if s == nil || s.db == nil {
		return transcript.ChangePage{}, errors.New("store: read-only session database is required")
	}
	if ctx == nil {
		return transcript.ChangePage{}, errors.New("store: read-only transcript changes context is required")
	}
	return queryTranscriptChanges(ctx, s.db, query)
}

func queryTranscriptPage(
	ctx context.Context,
	db *sql.DB,
	query transcript.PageQuery,
) (page transcript.Page, err error) {
	query, err = query.Normalize()
	if err != nil {
		return transcript.Page{}, err
	}
	tx, err := db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return transcript.Page{}, fmt.Errorf("store: begin transcript page read: %w", err)
	}
	defer func() {
		if rollbackErr := tx.Rollback(); rollbackErr != nil && !errors.Is(rollbackErr, sql.ErrTxDone) && err == nil {
			err = fmt.Errorf("store: close transcript page read: %w", rollbackErr)
		}
	}()
	state, err := loadProjectionState(ctx, tx)
	if err != nil {
		return transcript.Page{}, err
	}
	page.Generation = state.Generation
	if err := tx.QueryRowContext(
		ctx,
		"SELECT COALESCE(MAX(sequence), 0) FROM events",
	).Scan(&page.MaxSequence); err != nil {
		return transcript.Page{}, fmt.Errorf("store: query transcript max sequence: %w", err)
	}

	rows, err := queryMaterializedTranscriptPage(ctx, tx, query)
	if err != nil {
		return transcript.Page{}, fmt.Errorf("store: query transcript page: %w", err)
	}
	entries, err := scanMaterializedEntries(rows)
	if err != nil {
		return transcript.Page{}, err
	}
	if len(entries) > query.Limit {
		page.HasOlder = true
		entries = entries[:query.Limit]
	}
	slices.Reverse(entries)
	page.Entries = entries
	if page.HasOlder && len(entries) > 0 {
		page.NextBeforeSequence = entries[0].StartSequence
	}
	if err := tx.Commit(); err != nil {
		return transcript.Page{}, fmt.Errorf("store: commit transcript page read: %w", err)
	}
	return page, nil
}

func queryMaterializedTranscriptPage(
	ctx context.Context,
	tx *sql.Tx,
	query transcript.PageQuery,
) (*sql.Rows, error) {
	if query.BeforeSequence == 0 {
		return tx.QueryContext(ctx, `
			SELECT message_json, start_sequence, updated_sequence, event_type, marker_json
			FROM transcript_entries
			WHERE message_json IS NOT NULL
			ORDER BY start_sequence DESC
			LIMIT ?`, query.Limit+1)
	}
	return tx.QueryContext(ctx, `
		SELECT message_json, start_sequence, updated_sequence, event_type, marker_json
		FROM transcript_entries
		WHERE message_json IS NOT NULL AND start_sequence < ?
		ORDER BY start_sequence DESC
		LIMIT ?`, query.BeforeSequence, query.Limit+1)
}

func queryTranscriptChanges(
	ctx context.Context,
	db *sql.DB,
	query transcript.ChangeQuery,
) (page transcript.ChangePage, err error) {
	query, err = query.Normalize()
	if err != nil {
		return transcript.ChangePage{}, err
	}
	tx, err := db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return transcript.ChangePage{}, fmt.Errorf("store: begin transcript changes read: %w", err)
	}
	defer func() {
		if rollbackErr := tx.Rollback(); rollbackErr != nil && !errors.Is(rollbackErr, sql.ErrTxDone) && err == nil {
			err = fmt.Errorf("store: close transcript changes read: %w", rollbackErr)
		}
	}()
	state, err := loadProjectionState(ctx, tx)
	if err != nil {
		return transcript.ChangePage{}, err
	}
	page.Generation = state.Generation
	if err := tx.QueryRowContext(
		ctx,
		"SELECT COALESCE(MAX(sequence), 0) FROM events",
	).Scan(&page.MaxSequence); err != nil {
		return transcript.ChangePage{}, fmt.Errorf("store: query transcript changes max sequence: %w", err)
	}
	// One event can update its assigned entry and close at most one active
	// assistant. Rank paging keeps that bounded two-entry change atomic.
	rows, err := tx.QueryContext(ctx, `
		SELECT message_json, start_sequence, updated_sequence, event_type, marker_json, change_rank
		FROM (
			SELECT message_json, start_sequence, updated_sequence, event_type, marker_json,
				DENSE_RANK() OVER (ORDER BY updated_sequence ASC) AS change_rank
			FROM transcript_entries
			WHERE message_json IS NOT NULL AND updated_sequence > ?
		)
		WHERE change_rank <= ?
		ORDER BY updated_sequence ASC, start_sequence ASC`, query.AfterSequence, query.Limit+1)
	if err != nil {
		return transcript.ChangePage{}, fmt.Errorf("store: query transcript changes: %w", err)
	}
	entries, hasMore, err := scanMaterializedChangeEntries(rows, query.Limit)
	if err != nil {
		return transcript.ChangePage{}, err
	}
	page.HasMore = hasMore
	page.Entries = entries
	if len(entries) > 0 {
		page.NextAfter = entries[len(entries)-1].Sequence
	} else {
		page.NextAfter = query.AfterSequence
	}
	if err := tx.Commit(); err != nil {
		return transcript.ChangePage{}, fmt.Errorf("store: commit transcript changes read: %w", err)
	}
	return page, nil
}

func scanMaterializedEntries(rows *sql.Rows) (entries []transcript.Entry, err error) {
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			err = errors.Join(err, fmt.Errorf("store: close materialized transcript entries: %w", closeErr))
		}
	}()
	entries = make([]transcript.Entry, 0)
	for rows.Next() {
		entry, scanErr := scanMaterializedEntry(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		entries = append(entries, entry)
	}
	if rowsErr := rows.Err(); rowsErr != nil {
		return nil, fmt.Errorf("store: iterate materialized transcript entries: %w", rowsErr)
	}
	return entries, nil
}

func scanMaterializedChangeEntries(
	rows *sql.Rows,
	limit int,
) (entries []transcript.Entry, hasMore bool, err error) {
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			err = errors.Join(err, fmt.Errorf("store: close materialized transcript changes: %w", closeErr))
		}
	}()
	entries = make([]transcript.Entry, 0, limit)
	for rows.Next() {
		entry, rank, scanErr := scanMaterializedChangeEntry(rows)
		if scanErr != nil {
			return nil, false, scanErr
		}
		if rank > limit {
			hasMore = true
			continue
		}
		entries = append(entries, entry)
	}
	if rowsErr := rows.Err(); rowsErr != nil {
		return nil, false, fmt.Errorf("store: iterate materialized transcript changes: %w", rowsErr)
	}
	return entries, hasMore, nil
}

func scanMaterializedChangeEntry(scanner rowScanner) (transcript.Entry, int, error) {
	var entry transcript.Entry
	var messageJSON string
	var markerJSON sql.NullString
	var rank int
	if err := scanner.Scan(
		&messageJSON,
		&entry.StartSequence,
		&entry.Sequence,
		&entry.EventType,
		&markerJSON,
		&rank,
	); err != nil {
		return transcript.Entry{}, 0, fmt.Errorf("store: scan materialized transcript change: %w", err)
	}
	if err := decodeMaterializedEntry(&entry, messageJSON, markerJSON); err != nil {
		return transcript.Entry{}, 0, err
	}
	return entry, rank, nil
}

func scanMaterializedEntry(scanner rowScanner) (transcript.Entry, error) {
	var entry transcript.Entry
	var messageJSON string
	var markerJSON sql.NullString
	if err := scanner.Scan(
		&messageJSON,
		&entry.StartSequence,
		&entry.Sequence,
		&entry.EventType,
		&markerJSON,
	); err != nil {
		return transcript.Entry{}, fmt.Errorf("store: scan materialized transcript entry: %w", err)
	}
	if err := decodeMaterializedEntry(&entry, messageJSON, markerJSON); err != nil {
		return transcript.Entry{}, err
	}
	return entry, nil
}

func decodeMaterializedEntry(entry *transcript.Entry, messageJSON string, markerJSON sql.NullString) error {
	if err := json.Unmarshal([]byte(messageJSON), &entry.Message); err != nil {
		return fmt.Errorf(
			"%w: decode message at sequence %d: %v",
			transcript.ErrProjectionCorrupt,
			entry.StartSequence,
			err,
		)
	}
	if markerJSON.Valid {
		var marker transcript.Marker
		if err := json.Unmarshal([]byte(markerJSON.String), &marker); err != nil {
			return fmt.Errorf(
				"%w: decode marker at sequence %d: %v",
				transcript.ErrProjectionCorrupt,
				entry.StartSequence,
				err,
			)
		}
		entry.Marker = &marker
	}
	return nil
}
