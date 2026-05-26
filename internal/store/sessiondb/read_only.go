package sessiondb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"github.com/compozy/agh/internal/store"
)

const (
	readOnlyOpenMaxAttempts   = 15
	readOnlyOpenMinRetryDelay = 20 * time.Millisecond
	readOnlyOpenMaxRetryDelay = 150 * time.Millisecond
)

// ReadOnlySessionDB opens an existing per-session events database for queries
// without creating, migrating, checkpointing, or otherwise mutating it.
type ReadOnlySessionDB struct {
	db        *sql.DB
	sessionID string
}

var _ store.EventRecorder = (*ReadOnlySessionDB)(nil)

// OpenSessionDBReadOnly opens an existing per-session events database in
// SQLite read-only mode. It intentionally fails for missing paths instead of
// creating a fresh database during stale transcript/event reads.
func OpenSessionDBReadOnly(ctx context.Context, sessionID string, path string) (*ReadOnlySessionDB, error) {
	return openSessionDBReadOnlyWithRetry(ctx, sessionID, path, openSessionDBReadOnlyOnce, store.IsSQLiteBusy)
}

type readOnlySessionDBOpener func(context.Context, string, string) (*ReadOnlySessionDB, error)

func openSessionDBReadOnlyWithRetry(
	ctx context.Context,
	sessionID string,
	path string,
	opener readOnlySessionDBOpener,
	retryable func(error) bool,
) (*ReadOnlySessionDB, error) {
	if opener == nil {
		return nil, errors.New("store: read-only session database opener is required")
	}
	if retryable == nil {
		retryable = func(error) bool { return false }
	}

	var lastErr error
	for attempt := 1; attempt <= readOnlyOpenMaxAttempts; attempt++ {
		reader, err := opener(ctx, sessionID, path)
		if err == nil {
			return reader, nil
		}
		lastErr = err
		if !retryable(err) || attempt == readOnlyOpenMaxAttempts {
			return nil, err
		}
		if waitErr := waitForReadOnlyOpenRetry(ctx, readOnlyOpenRetryDelay(attempt)); waitErr != nil {
			return nil, errors.Join(err, waitErr)
		}
	}
	return nil, lastErr
}

func openSessionDBReadOnlyOnce(ctx context.Context, sessionID string, path string) (*ReadOnlySessionDB, error) {
	if ctx == nil {
		return nil, errors.New("store: open read-only session database context is required")
	}
	cleanSessionID := strings.TrimSpace(sessionID)
	if cleanSessionID == "" {
		return nil, errors.New("store: read-only session database session id is required")
	}
	cleanPath := strings.TrimSpace(path)
	if cleanPath == "" {
		return nil, errors.New("store: read-only session database path is required")
	}

	db, err := sql.Open("sqlite", readOnlySessionSQLiteDSN(cleanPath))
	if err != nil {
		return nil, fmt.Errorf("store: open read-only session database %q: %w", cleanPath, err)
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	if err := db.PingContext(ctx); err != nil {
		return nil, closeReadOnlySessionDBAfterOpenError(
			db,
			fmt.Errorf("store: ping read-only session database %q: %w", cleanPath, err),
		)
	}
	if _, err := db.ExecContext(ctx, "PRAGMA query_only = ON"); err != nil {
		return nil, closeReadOnlySessionDBAfterOpenError(
			db,
			fmt.Errorf("store: guard read-only session database %q: %w", cleanPath, err),
		)
	}

	return &ReadOnlySessionDB{db: db, sessionID: cleanSessionID}, nil
}

func readOnlySessionSQLiteDSN(path string) string {
	u := url.URL{
		Scheme: "file",
		Path:   filepath.ToSlash(path),
	}
	query := u.Query()
	query.Set("mode", "ro")
	query.Add("_pragma", fmt.Sprintf("busy_timeout(%d)", store.DefaultSQLiteBusyTimeoutMS))
	u.RawQuery = query.Encode()
	return u.String()
}

func readOnlyOpenRetryDelay(attempt int) time.Duration {
	if attempt <= 0 {
		return readOnlyOpenMinRetryDelay
	}
	delay := time.Duration(attempt) * readOnlyOpenMinRetryDelay
	if delay > readOnlyOpenMaxRetryDelay {
		return readOnlyOpenMaxRetryDelay
	}
	return delay
}

func waitForReadOnlyOpenRetry(ctx context.Context, delay time.Duration) error {
	if ctx == nil {
		return errors.New("store: read-only session database retry context is required")
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return fmt.Errorf("store: wait for read-only session database retry: %w", ctx.Err())
	case <-timer.C:
		return nil
	}
}

func closeReadOnlySessionDBAfterOpenError(db *sql.DB, openErr error) error {
	if db == nil {
		return openErr
	}
	if closeErr := db.Close(); closeErr != nil {
		return errors.Join(
			openErr,
			fmt.Errorf("store: close read-only session database after open failure: %w", closeErr),
		)
	}
	return openErr
}

func (s *ReadOnlySessionDB) Record(context.Context, store.SessionEvent) error {
	return errors.New("store: read-only session database cannot record events")
}

func (s *ReadOnlySessionDB) RecordTokenUsage(context.Context, store.TokenUsage) error {
	return errors.New("store: read-only session database cannot record token usage")
}

// Query returns events filtered by the supplied options.
func (s *ReadOnlySessionDB) Query(
	ctx context.Context,
	query store.EventQuery,
) (events []store.SessionEvent, err error) {
	if s == nil || s.db == nil {
		return nil, errors.New("store: read-only session database is required")
	}
	if ctx == nil {
		return nil, errors.New("store: query read-only session database context is required")
	}
	if err := query.Validate(); err != nil {
		return nil, err
	}

	baseQuery := `SELECT id, sequence, turn_id, type, agent_name, content, timestamp FROM events`
	where, args := store.BuildClauses(
		store.StringClause("type", query.Type),
		store.StringClause("agent_name", query.AgentName),
		store.StringClause("turn_id", query.TurnID),
		store.TimeClause("timestamp", ">=", query.Since),
		store.Int64Clause("sequence", ">", query.AfterSequence),
	)
	baseQuery = store.AppendWhere(baseQuery, where)

	sqlQuery := baseQuery
	if query.Limit > 0 {
		sqlQuery = `SELECT id, sequence, turn_id, type, agent_name, content, timestamp
			FROM (` + baseQuery + ` ORDER BY sequence DESC LIMIT ?) AS recent_events
			ORDER BY sequence ASC`
		args = append(args, query.Limit)
	} else {
		sqlQuery += " ORDER BY sequence ASC"
	}

	rows, err := s.db.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("store: query read-only session events: %w", err)
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("store: close read-only session event rows: %w", closeErr)
		}
	}()

	events = make([]store.SessionEvent, 0)
	for rows.Next() {
		event, scanErr := s.scanSessionEvent(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: iterate read-only session events: %w", err)
	}

	return events, nil
}

// History returns ordered session events grouped by turn id.
func (s *ReadOnlySessionDB) History(ctx context.Context, query store.EventQuery) ([]store.TurnHistory, error) {
	events, err := s.Query(ctx, query)
	if err != nil {
		return nil, err
	}

	turns := make([]store.TurnHistory, 0)
	indexByTurnID := make(map[string]int, len(events))
	for _, event := range events {
		if idx, ok := indexByTurnID[event.TurnID]; ok {
			turns[idx].Events = append(turns[idx].Events, event)
			continue
		}
		indexByTurnID[event.TurnID] = len(turns)
		turns = append(turns, store.TurnHistory{
			TurnID: event.TurnID,
			Events: []store.SessionEvent{event},
		})
	}

	return turns, nil
}

// Close closes the read-only database handle without checkpointing.
func (s *ReadOnlySessionDB) Close(ctx context.Context) error {
	if s == nil || s.db == nil {
		return nil
	}
	if ctx == nil {
		return errors.New("store: close read-only session database context is required")
	}
	if err := s.db.Close(); err != nil {
		return fmt.Errorf("store: close read-only session database: %w", err)
	}
	s.db = nil
	return nil
}

func (s *ReadOnlySessionDB) scanSessionEvent(scanner rowScanner) (store.SessionEvent, error) {
	var (
		event     store.SessionEvent
		timestamp string
	)
	if err := scanner.Scan(
		&event.ID,
		&event.Sequence,
		&event.TurnID,
		&event.Type,
		&event.AgentName,
		&event.Content,
		&timestamp,
	); err != nil {
		return store.SessionEvent{}, fmt.Errorf("store: scan read-only session event: %w", err)
	}

	parsed, err := store.ParseTimestamp(timestamp)
	if err != nil {
		return store.SessionEvent{}, err
	}
	event.Timestamp = parsed
	event.SessionID = s.sessionID
	return event, nil
}
