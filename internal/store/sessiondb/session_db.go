package sessiondb

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	hookspkg "github.com/pedronauck/agh/internal/hooks"
	"github.com/pedronauck/agh/internal/store"
)

const (
	defaultWriteBufferSize         = 256
	defaultDrainTimeout            = 5 * time.Second
	canonicalEventSchema           = "agh.session.event.v1"
	sessionVacuumMinBytes          = 4 << 20
	sessionVacuumMinRatio          = 4
	sessionWALAutoCheckpointPragma = "wal_autocheckpoint(0)"
)

var sessionSchemaStatements = []string{
	`CREATE TABLE IF NOT EXISTS events (
		id         TEXT PRIMARY KEY,
		sequence   INTEGER NOT NULL,
		turn_id    TEXT NOT NULL,
		type       TEXT NOT NULL,
		agent_name TEXT NOT NULL,
		content    TEXT NOT NULL,
		timestamp  TEXT NOT NULL
	);`,
	`CREATE INDEX IF NOT EXISTS idx_events_type ON events(type);`,
	`CREATE INDEX IF NOT EXISTS idx_events_timestamp ON events(timestamp);`,
	`CREATE INDEX IF NOT EXISTS idx_events_sequence ON events(sequence);`,
	`CREATE INDEX IF NOT EXISTS idx_events_turn ON events(turn_id);`,
	`CREATE TABLE IF NOT EXISTS token_usage (
		turn_id            TEXT PRIMARY KEY,
		input_tokens       INTEGER,
		output_tokens      INTEGER,
		total_tokens       INTEGER,
		thought_tokens     INTEGER,
		cache_read_tokens  INTEGER,
		cache_write_tokens INTEGER,
		context_used       INTEGER,
		context_size       INTEGER,
		cost_amount        REAL,
		cost_currency      TEXT,
		timestamp          TEXT NOT NULL
	);`,
	`CREATE INDEX IF NOT EXISTS idx_usage_timestamp ON token_usage(timestamp);`,
	`CREATE TABLE IF NOT EXISTS hook_runs (
		id             TEXT PRIMARY KEY,
		hook_name      TEXT NOT NULL,
		event          TEXT NOT NULL,
		source         TEXT NOT NULL,
		mode           TEXT NOT NULL,
		duration_ns    INTEGER NOT NULL,
		outcome        TEXT NOT NULL,
		dispatch_depth INTEGER NOT NULL,
		patch_applied  TEXT,
		error          TEXT,
		required       INTEGER NOT NULL DEFAULT 0,
		recorded_at    TEXT NOT NULL
	);`,
	`CREATE INDEX IF NOT EXISTS idx_hook_runs_event ON hook_runs(event);`,
	`CREATE INDEX IF NOT EXISTS idx_hook_runs_recorded_at ON hook_runs(recorded_at);`,
}

var sessionSchemaMigrations = []store.Migration{
	{
		Version:    1,
		Name:       "create_session_schema",
		Statements: sessionSchemaStatements,
	},
	{
		Version:  2,
		Name:     "strip_canonical_event_raw_payloads",
		Up:       stripCanonicalEventRawPayloads,
		Checksum: "2026-04-25-strip-canonical-event-raw-payloads",
	},
}

const (
	sessionStateOpen int32 = iota
	sessionStateClosing
	sessionStateClosed
)

type sessionWriteKind int

const (
	sessionWriteEvent sessionWriteKind = iota + 1
	sessionWriteUsage
	sessionWriteHookRun
)

type sessionWriteRequest struct {
	ctx    context.Context
	kind   sessionWriteKind
	event  store.SessionEvent
	usage  store.TokenUsage
	hook   hookspkg.HookRunRecord
	result chan error
}

type sessionShutdownRequest struct {
	ctx    context.Context
	result chan error
}

// SessionDB owns a per-session SQLite database and its dedicated writer loop.
type SessionDB struct {
	db         *sql.DB
	path       string
	sessionID  string
	writeCh    chan sessionWriteRequest
	shutdownCh chan sessionShutdownRequest
	writerDone chan struct{}
	writerCtx  context.Context
	cancel     context.CancelFunc

	acceptMu sync.RWMutex
	state    atomic.Int32

	drainTimeout time.Duration
	now          func() time.Time
	nextSequence int64
}

var _ store.EventRecorder = (*SessionDB)(nil)

// OpenSessionDB opens or creates the per-session events database at path.
func OpenSessionDB(ctx context.Context, sessionID string, path string) (*SessionDB, error) {
	if ctx == nil {
		return nil, errors.New("store: open session database context is required")
	}
	if strings.TrimSpace(sessionID) == "" {
		return nil, errors.New("store: session database session id is required")
	}

	db, err := openSessionSQLite(ctx, path)
	if err != nil {
		return nil, err
	}

	nextSequence, err := currentMaxSequence(ctx, db)
	if err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("store: load current sequence for %q: %w", path, err)
	}

	sessionDB := &SessionDB{
		db:           db,
		path:         strings.TrimSpace(path),
		sessionID:    strings.TrimSpace(sessionID),
		writeCh:      make(chan sessionWriteRequest, defaultWriteBufferSize),
		shutdownCh:   make(chan sessionShutdownRequest, 1),
		writerDone:   make(chan struct{}),
		drainTimeout: defaultDrainTimeout,
		now: func() time.Time {
			return time.Now().UTC()
		},
		nextSequence: nextSequence,
	}
	sessionDB.writerCtx, sessionDB.cancel = context.WithCancel(context.Background())
	sessionDB.state.Store(sessionStateOpen)

	go func() {
		defer close(sessionDB.writerDone)
		sessionDB.writerLoop()
	}()

	return sessionDB, nil
}

// Path reports the on-disk path for the database file.
func (s *SessionDB) Path() string {
	if s == nil {
		return ""
	}
	return s.path
}

// SessionID reports the owning session identifier for the database.
func (s *SessionDB) SessionID() string {
	if s == nil {
		return ""
	}
	return s.sessionID
}

// Record appends a session event using the dedicated writer goroutine.
func (s *SessionDB) Record(ctx context.Context, event store.SessionEvent) error {
	if s == nil {
		return errors.New("store: session database is required")
	}
	if ctx == nil {
		return errors.New("store: record event context is required")
	}
	if err := event.Validate(); err != nil {
		return err
	}
	if event.SessionID != "" && event.SessionID != s.sessionID {
		return fmt.Errorf("store: event session id %q does not match session database %q", event.SessionID, s.sessionID)
	}
	event.SessionID = s.sessionID

	return s.enqueueWrite(ctx, sessionWriteRequest{
		ctx:    ctx,
		kind:   sessionWriteEvent,
		event:  event,
		result: make(chan error, 1),
	})
}

// RecordTokenUsage stores or merges per-turn usage data for the session.
func (s *SessionDB) RecordTokenUsage(ctx context.Context, usage store.TokenUsage) error {
	if s == nil {
		return errors.New("store: session database is required")
	}
	if ctx == nil {
		return errors.New("store: record token usage context is required")
	}
	if err := usage.Validate(); err != nil {
		return err
	}

	return s.enqueueWrite(ctx, sessionWriteRequest{
		ctx:    ctx,
		kind:   sessionWriteUsage,
		usage:  usage,
		result: make(chan error, 1),
	})
}

// RecordHookRun stores one hook execution audit record in the per-session store.
func (s *SessionDB) RecordHookRun(ctx context.Context, record hookspkg.HookRunRecord) error {
	if s == nil {
		return errors.New("store: session database is required")
	}
	if ctx == nil {
		return errors.New("store: record hook run context is required")
	}

	return s.enqueueWrite(ctx, sessionWriteRequest{
		ctx:    ctx,
		kind:   sessionWriteHookRun,
		hook:   cloneHookRunRecord(record),
		result: make(chan error, 1),
	})
}

// QueryHookRuns returns persisted hook execution records filtered by the supplied options.
func (s *SessionDB) QueryHookRuns(ctx context.Context, query store.HookRunQuery) ([]hookspkg.HookRunRecord, error) {
	if s == nil {
		return nil, errors.New("store: session database is required")
	}
	if ctx == nil {
		return nil, errors.New("store: query hook runs context is required")
	}
	if err := query.Validate(); err != nil {
		return nil, err
	}
	if strings.TrimSpace(query.SessionID) != "" && strings.TrimSpace(query.SessionID) != s.sessionID {
		return nil, fmt.Errorf(
			"store: hook run query session id %q does not match session database %q",
			query.SessionID,
			s.sessionID,
		)
	}
	if event := strings.TrimSpace(query.Event); event != "" {
		if err := hookspkg.HookEvent(event).Validate(); err != nil {
			return nil, err
		}
	}
	if query.Outcome != "" {
		if err := query.Outcome.Validate(); err != nil {
			return nil, err
		}
	}
	s.acceptMu.RLock()
	defer s.acceptMu.RUnlock()
	if s.state.Load() != sessionStateOpen {
		return nil, store.ErrClosed
	}

	baseQuery := `SELECT
		rowid, hook_name, event, source, mode, duration_ns, outcome,
		dispatch_depth, patch_applied, error, required, recorded_at
		FROM hook_runs`
	where, args := store.BuildClauses(
		store.StringClause("event", query.Event),
		store.StringClause("outcome", string(query.Outcome)),
		store.TimeClause("recorded_at", ">=", query.Since),
	)
	baseQuery = store.AppendWhere(baseQuery, where)

	sqlQuery := baseQuery
	if query.Limit > 0 {
		sqlQuery = `SELECT
				rowid, hook_name, event, source, mode, duration_ns, outcome,
				dispatch_depth, patch_applied, error, required, recorded_at
				FROM (` + baseQuery + ` ORDER BY recorded_at DESC, rowid DESC LIMIT ?) AS recent_hook_runs
				ORDER BY recorded_at ASC, rowid ASC`
		args = append(args, query.Limit)
	} else {
		sqlQuery += " ORDER BY recorded_at ASC, rowid ASC"
	}

	rows, err := s.db.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("store: query hook runs: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	records := make([]hookspkg.HookRunRecord, 0)
	for rows.Next() {
		record, scanErr := s.scanHookRunRecord(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: iterate hook runs: %w", err)
	}

	return records, nil
}

// Query returns events filtered by the supplied options.
func (s *SessionDB) Query(ctx context.Context, query store.EventQuery) ([]store.SessionEvent, error) {
	if s == nil {
		return nil, errors.New("store: session database is required")
	}
	if ctx == nil {
		return nil, errors.New("store: query events context is required")
	}
	if err := query.Validate(); err != nil {
		return nil, err
	}
	s.acceptMu.RLock()
	defer s.acceptMu.RUnlock()
	if s.state.Load() != sessionStateOpen {
		return nil, store.ErrClosed
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
		return nil, fmt.Errorf("store: query session events: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	events := make([]store.SessionEvent, 0)
	for rows.Next() {
		event, scanErr := s.scanSessionEvent(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: iterate session events: %w", err)
	}

	return events, nil
}

// History returns ordered session events grouped by turn id.
func (s *SessionDB) History(ctx context.Context, query store.EventQuery) ([]store.TurnHistory, error) {
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

// Close drains queued writes, checkpoints the WAL, and closes the database.
func (s *SessionDB) Close(ctx context.Context) error {
	if s == nil {
		return nil
	}
	if ctx == nil {
		return errors.New("store: close session database context is required")
	}
	if !s.state.CompareAndSwap(sessionStateOpen, sessionStateClosing) {
		if s.state.Load() == sessionStateClosed {
			return nil
		}
		return store.ErrClosed
	}

	drainCtx, cancel := context.WithTimeout(ctx, s.drainTimeout)
	defer cancel()
	if s.cancel != nil {
		defer s.cancel()
	}

	s.acceptMu.Lock()
	resultCh := make(chan error, 1)
	s.shutdownCh <- sessionShutdownRequest{
		ctx:    drainCtx,
		result: resultCh,
	}
	s.acceptMu.Unlock()

	writerErr := waitForShutdownResult(drainCtx, resultCh)
	writerExitErr := waitForWriterExit(drainCtx, s.writerDone)
	checkpointErr := store.Checkpoint(drainCtx, s.db)
	closeErr := s.db.Close()

	s.state.Store(sessionStateClosed)

	return errors.Join(writerErr, writerExitErr, checkpointErr, closeErr)
}

func (s *SessionDB) enqueueWrite(ctx context.Context, req sessionWriteRequest) error {
	s.acceptMu.RLock()
	defer s.acceptMu.RUnlock()

	if s.state.Load() != sessionStateOpen {
		return store.ErrClosed
	}

	select {
	case s.writeCh <- req:
	case <-ctx.Done():
		return fmt.Errorf("store: enqueue session write: %w", ctx.Err())
	}

	select {
	case err := <-req.result:
		return err
	case <-ctx.Done():
		return fmt.Errorf("store: wait for session write completion: %w", ctx.Err())
	}
}

func (s *SessionDB) writerLoop() {
	for {
		select {
		case req := <-s.writeCh:
			req.result <- s.executeWrite(req)
		case shutdown := <-s.shutdownCh:
			shutdown.result <- s.drainWrites(shutdown.ctx)
			return
		case <-s.writerCtx.Done():
			return
		}
	}
}

func (s *SessionDB) drainWrites(ctx context.Context) error {
	var drainErr error

	for {
		select {
		case <-ctx.Done():
			return errors.Join(drainErr, fmt.Errorf("%w: %w", store.ErrDrainTimeout, ctx.Err()))
		case req := <-s.writeCh:
			err := s.executeWrite(req)
			req.result <- err
			if err != nil {
				drainErr = errors.Join(drainErr, err)
			}
		default:
			return drainErr
		}
	}
}

func (s *SessionDB) executeWrite(req sessionWriteRequest) error {
	if err := req.ctx.Err(); err != nil {
		return fmt.Errorf("store: session write canceled before execution: %w", err)
	}

	switch req.kind {
	case sessionWriteEvent:
		return s.writeEvent(req.ctx, req.event)
	case sessionWriteUsage:
		return s.writeTokenUsage(req.ctx, req.usage)
	case sessionWriteHookRun:
		return s.writeHookRun(req.ctx, req.hook)
	default:
		return fmt.Errorf("store: unsupported session write kind %d", req.kind)
	}
}

func (s *SessionDB) writeEvent(ctx context.Context, event store.SessionEvent) error {
	if strings.TrimSpace(event.ID) == "" {
		event.ID = store.NewID("ev")
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = s.now()
	}

	s.nextSequence++
	event.Sequence = s.nextSequence

	if _, err := s.db.ExecContext(
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
		s.nextSequence--
		return fmt.Errorf("store: insert session event: %w", err)
	}

	return nil
}

func (s *SessionDB) writeTokenUsage(ctx context.Context, usage store.TokenUsage) error {
	if usage.Timestamp.IsZero() {
		usage.Timestamp = s.now()
	}

	if _, err := s.db.ExecContext(
		ctx,
		`INSERT INTO token_usage (
			turn_id, input_tokens, output_tokens, total_tokens, thought_tokens,
			cache_read_tokens, cache_write_tokens, context_used, context_size,
			cost_amount, cost_currency, timestamp
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(turn_id) DO UPDATE SET
			input_tokens = COALESCE(excluded.input_tokens, token_usage.input_tokens),
			output_tokens = COALESCE(excluded.output_tokens, token_usage.output_tokens),
			total_tokens = COALESCE(excluded.total_tokens, token_usage.total_tokens),
			thought_tokens = COALESCE(excluded.thought_tokens, token_usage.thought_tokens),
			cache_read_tokens = COALESCE(excluded.cache_read_tokens, token_usage.cache_read_tokens),
			cache_write_tokens = COALESCE(excluded.cache_write_tokens, token_usage.cache_write_tokens),
			context_used = COALESCE(excluded.context_used, token_usage.context_used),
			context_size = COALESCE(excluded.context_size, token_usage.context_size),
			cost_amount = COALESCE(excluded.cost_amount, token_usage.cost_amount),
			cost_currency = COALESCE(excluded.cost_currency, token_usage.cost_currency),
			timestamp = excluded.timestamp`,
		usage.TurnID,
		store.NullableInt64(usage.InputTokens),
		store.NullableInt64(usage.OutputTokens),
		store.NullableInt64(usage.TotalTokens),
		store.NullableInt64(usage.ThoughtTokens),
		store.NullableInt64(usage.CacheReadTokens),
		store.NullableInt64(usage.CacheWriteTokens),
		store.NullableInt64(usage.ContextUsed),
		store.NullableInt64(usage.ContextSize),
		store.NullableFloat64(usage.CostAmount),
		store.NullableStringPointer(usage.CostCurrency),
		store.FormatTimestamp(usage.Timestamp),
	); err != nil {
		return fmt.Errorf("store: upsert token usage: %w", err)
	}

	return nil
}

func (s *SessionDB) writeHookRun(ctx context.Context, record hookspkg.HookRunRecord) error {
	if strings.TrimSpace(record.HookName) == "" {
		return errors.New("store: hook run hook name is required")
	}
	if err := record.Event.Validate(); err != nil {
		return err
	}
	if err := record.Source.Validate(); err != nil {
		return err
	}
	if err := record.Mode.Validate(); err != nil {
		return err
	}
	if record.RecordedAt.IsZero() {
		record.RecordedAt = s.now()
	}

	id := store.NewID("hook")
	if _, err := s.db.ExecContext(
		ctx,
		`INSERT INTO hook_runs (
			id, hook_name, event, source, mode, duration_ns, outcome, dispatch_depth,
			patch_applied, error, required, recorded_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		id,
		record.HookName,
		record.Event.String(),
		record.Source.String(),
		string(record.Mode),
		record.Duration.Nanoseconds(),
		string(record.Outcome),
		record.DispatchDepth,
		store.NullableString(rawJSONText(record.PatchApplied)),
		store.NullableString(record.Error),
		boolToSQLite(record.Required),
		store.FormatTimestamp(record.RecordedAt),
	); err != nil {
		return fmt.Errorf("store: insert hook run: %w", err)
	}

	return nil
}

func (s *SessionDB) scanSessionEvent(scanner rowScanner) (store.SessionEvent, error) {
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
		return store.SessionEvent{}, fmt.Errorf("store: scan session event: %w", err)
	}

	parsed, err := store.ParseTimestamp(timestamp)
	if err != nil {
		return store.SessionEvent{}, err
	}
	event.Timestamp = parsed
	event.SessionID = s.sessionID
	return event, nil
}

func (s *SessionDB) scanHookRunRecord(scanner rowScanner) (hookspkg.HookRunRecord, error) {
	var (
		record        hookspkg.HookRunRecord
		rowID         int64
		event         string
		source        string
		mode          string
		durationNS    int64
		outcome       string
		patchApplied  sql.NullString
		recordError   sql.NullString
		required      int64
		recordedAtRaw string
	)

	if err := scanner.Scan(
		&rowID,
		&record.HookName,
		&event,
		&source,
		&mode,
		&durationNS,
		&outcome,
		&record.DispatchDepth,
		&patchApplied,
		&recordError,
		&required,
		&recordedAtRaw,
	); err != nil {
		return hookspkg.HookRunRecord{}, fmt.Errorf("store: scan hook run: %w", err)
	}

	record.Event = hookspkg.HookEvent(strings.TrimSpace(event))
	if err := record.Event.Validate(); err != nil {
		return hookspkg.HookRunRecord{}, err
	}
	if err := record.Source.UnmarshalText([]byte(strings.TrimSpace(source))); err != nil {
		return hookspkg.HookRunRecord{}, err
	}
	record.Mode = hookspkg.HookMode(strings.TrimSpace(mode))
	record.Duration = time.Duration(durationNS)
	record.Outcome = hookspkg.HookRunOutcome(strings.TrimSpace(outcome))
	record.Required = required != 0
	record.Error = strings.TrimSpace(recordError.String)
	if patchApplied.Valid && strings.TrimSpace(patchApplied.String) != "" {
		record.PatchApplied = json.RawMessage(patchApplied.String)
	}

	recordedAt, err := store.ParseTimestamp(recordedAtRaw)
	if err != nil {
		return hookspkg.HookRunRecord{}, err
	}
	record.RecordedAt = recordedAt
	_ = rowID
	return cloneHookRunRecord(record), nil
}

func currentMaxSequence(ctx context.Context, db *sql.DB) (int64, error) {
	var sequence int64
	if err := db.QueryRowContext(ctx, "SELECT COALESCE(MAX(sequence), 0) FROM events").Scan(&sequence); err != nil {
		return 0, err
	}
	return sequence, nil
}

func waitForShutdownResult(ctx context.Context, resultCh <-chan error) error {
	select {
	case err := <-resultCh:
		return err
	case <-ctx.Done():
		return fmt.Errorf("%w: %w", store.ErrDrainTimeout, ctx.Err())
	}
}

func waitForWriterExit(ctx context.Context, done <-chan struct{}) error {
	if done == nil {
		return nil
	}
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("%w: %w", store.ErrDrainTimeout, ctx.Err())
	}
}

type rowScanner interface {
	Scan(dest ...any) error
}

func openSessionSQLite(ctx context.Context, path string) (*sql.DB, error) {
	return openSessionSQLiteWithVacuum(ctx, path, vacuumSessionSQLite)
}

type sessionVacuumFunc func(context.Context, *sql.DB) error

func openSessionSQLiteWithVacuum(
	ctx context.Context,
	path string,
	vacuumFn sessionVacuumFunc,
) (*sql.DB, error) {
	return store.OpenSQLiteDatabaseWithPragmas(
		ctx,
		path,
		[]string{sessionWALAutoCheckpointPragma},
		func(ctx context.Context, db *sql.DB) error {
			if err := store.RunMigrations(ctx, db, sessionSchemaMigrations); err != nil {
				return err
			}
			if vacuumFn == nil {
				return nil
			}
			if err := vacuumFn(ctx, db); err != nil {
				slog.Default().WarnContext(
					ctx,
					"store: skip session sqlite vacuum after non-fatal failure",
					"path",
					path,
					"error",
					err,
				)
			}
			return nil
		},
	)
}

func stripCanonicalEventRawPayloads(ctx context.Context, tx *sql.Tx) error {
	if ctx == nil {
		return errors.New("store: session raw-strip migration context is required")
	}
	if tx == nil {
		return errors.New("store: session raw-strip migration transaction is required")
	}

	if _, err := tx.ExecContext(
		ctx,
		`UPDATE events
		 SET content = json_remove(content, '$.raw')
		 WHERE json_valid(content) = 1
		   AND json_extract(content, '$.schema') = ?
		   AND json_type(content, '$.raw') IS NOT NULL`,
		canonicalEventSchema,
	); err != nil {
		return fmt.Errorf("store: strip canonical session event raw payloads: %w", err)
	}
	return nil
}

type sqlitePageStats struct {
	pageCount     int64
	pageSize      int64
	freelistCount int64
}

func vacuumSessionSQLite(ctx context.Context, db *sql.DB) error {
	stats, err := loadSQLitePageStats(ctx, db)
	if err != nil {
		return err
	}
	if !shouldVacuumSessionSQLite(stats) {
		return nil
	}
	if _, err := db.ExecContext(ctx, "VACUUM"); err != nil {
		return fmt.Errorf("store: vacuum session sqlite database: %w", err)
	}
	return nil
}

func loadSQLitePageStats(ctx context.Context, db *sql.DB) (sqlitePageStats, error) {
	if ctx == nil {
		return sqlitePageStats{}, errors.New("store: sqlite page stats context is required")
	}
	if db == nil {
		return sqlitePageStats{}, errors.New("store: sqlite page stats database is required")
	}

	var stats sqlitePageStats
	if err := db.QueryRowContext(ctx, "PRAGMA page_count").Scan(&stats.pageCount); err != nil {
		return sqlitePageStats{}, fmt.Errorf("store: query sqlite page_count: %w", err)
	}
	if err := db.QueryRowContext(ctx, "PRAGMA page_size").Scan(&stats.pageSize); err != nil {
		return sqlitePageStats{}, fmt.Errorf("store: query sqlite page_size: %w", err)
	}
	if err := db.QueryRowContext(ctx, "PRAGMA freelist_count").Scan(&stats.freelistCount); err != nil {
		return sqlitePageStats{}, fmt.Errorf("store: query sqlite freelist_count: %w", err)
	}
	return stats, nil
}

func shouldVacuumSessionSQLite(stats sqlitePageStats) bool {
	if stats.pageCount <= 0 || stats.pageSize <= 0 || stats.freelistCount <= 0 {
		return false
	}
	freeBytes := stats.freelistCount * stats.pageSize
	if freeBytes < sessionVacuumMinBytes {
		return false
	}
	return stats.freelistCount*sessionVacuumMinRatio >= stats.pageCount
}

func rawJSONText(raw json.RawMessage) string {
	return strings.TrimSpace(string(raw))
}

func boolToSQLite(value bool) int64 {
	if value {
		return 1
	}
	return 0
}

func cloneHookRunRecord(src hookspkg.HookRunRecord) hookspkg.HookRunRecord {
	cloned := src
	cloned.PatchApplied = cloneRawJSON(src.PatchApplied)
	return cloned
}

func cloneRawJSON(src json.RawMessage) json.RawMessage {
	if len(src) == 0 {
		return nil
	}
	return append(json.RawMessage(nil), src...)
}
