package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	_ "modernc.org/sqlite"
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
}

var globalSchemaStatements = []string{
	`CREATE TABLE IF NOT EXISTS workspaces (
		id            TEXT PRIMARY KEY,
		root_dir      TEXT NOT NULL UNIQUE,
		add_dirs      TEXT NOT NULL DEFAULT '[]',
		name          TEXT NOT NULL UNIQUE,
		default_agent TEXT DEFAULT '',
		created_at    TEXT NOT NULL,
		updated_at    TEXT NOT NULL
	);`,
	`CREATE INDEX IF NOT EXISTS idx_workspaces_name ON workspaces(name);`,
	`CREATE TABLE IF NOT EXISTS sessions (
		id             TEXT PRIMARY KEY,
		name           TEXT,
		agent_name     TEXT NOT NULL,
		workspace_id   TEXT NOT NULL REFERENCES workspaces(id),
		session_type   TEXT NOT NULL DEFAULT 'user',
		state          TEXT NOT NULL,
		acp_session_id TEXT,
		created_at     TEXT NOT NULL,
		updated_at     TEXT NOT NULL
	);`,
	`CREATE TABLE IF NOT EXISTS event_summaries (
		id         TEXT PRIMARY KEY,
		session_id TEXT NOT NULL REFERENCES sessions(id),
		type       TEXT NOT NULL,
		agent_name TEXT NOT NULL,
		summary    TEXT,
		timestamp  TEXT NOT NULL
	);`,
	`CREATE INDEX IF NOT EXISTS idx_summaries_session ON event_summaries(session_id);`,
	`CREATE INDEX IF NOT EXISTS idx_summaries_type ON event_summaries(type);`,
	`CREATE INDEX IF NOT EXISTS idx_summaries_timestamp ON event_summaries(timestamp);`,
	`CREATE TABLE IF NOT EXISTS token_stats (
		id            TEXT PRIMARY KEY,
		session_id    TEXT NOT NULL REFERENCES sessions(id),
		agent_name    TEXT NOT NULL,
		input_tokens  INTEGER,
		output_tokens INTEGER,
		total_tokens  INTEGER,
		total_cost    REAL,
		cost_currency TEXT,
		turn_count    INTEGER NOT NULL DEFAULT 0,
		updated_at    TEXT NOT NULL
	);`,
	`CREATE INDEX IF NOT EXISTS idx_token_stats_session ON token_stats(session_id);`,
	`CREATE UNIQUE INDEX IF NOT EXISTS idx_token_stats_session_agent ON token_stats(session_id, agent_name);`,
	`CREATE TABLE IF NOT EXISTS permission_log (
		id          TEXT PRIMARY KEY,
		session_id  TEXT NOT NULL REFERENCES sessions(id),
		agent_name  TEXT NOT NULL,
		action      TEXT NOT NULL,
		resource    TEXT NOT NULL,
		decision    TEXT NOT NULL,
		policy_used TEXT NOT NULL,
		timestamp   TEXT NOT NULL
	);`,
	`CREATE INDEX IF NOT EXISTS idx_perm_session ON permission_log(session_id);`,
}

func openSessionSQLite(ctx context.Context, path string) (*sql.DB, error) {
	return openSQLiteDatabase(ctx, path, func(ctx context.Context, db *sql.DB) error {
		return ensureSchema(ctx, db, sessionSchemaStatements)
	})
}

func openGlobalSQLite(ctx context.Context, path string) (*sql.DB, error) {
	return openSQLiteDatabase(ctx, path, func(ctx context.Context, db *sql.DB) error {
		if err := migrateGlobalSchema(ctx, db); err != nil {
			return err
		}
		return ensureSchema(ctx, db, globalSchemaStatements)
	})
}

func openSQLiteDatabase(ctx context.Context, path string, initialize func(context.Context, *sql.DB) error) (*sql.DB, error) {
	cleanPath := strings.TrimSpace(path)
	if cleanPath == "" {
		return nil, errors.New("store: database path is required")
	}
	if err := os.MkdirAll(filepath.Dir(cleanPath), 0o755); err != nil {
		return nil, fmt.Errorf("store: create database directory for %q: %w", cleanPath, err)
	}

	db, err := openSQLiteDatabaseOnce(ctx, cleanPath, initialize)
	if err == nil {
		return db, nil
	}
	if !shouldRecoverSQLite(err) {
		return nil, err
	}
	if _, statErr := os.Stat(cleanPath); statErr != nil {
		return nil, err
	}
	if _, recoverErr := recoverSQLiteDatabase(cleanPath); recoverErr != nil {
		return nil, errors.Join(err, fmt.Errorf("store: recover sqlite database %q: %w", cleanPath, recoverErr))
	}

	db, reopenErr := openSQLiteDatabaseOnce(ctx, cleanPath, initialize)
	if reopenErr != nil {
		return nil, errors.Join(err, fmt.Errorf("store: reopen sqlite database %q after recovery: %w", cleanPath, reopenErr))
	}
	return db, nil
}

func openSQLiteDatabaseOnce(ctx context.Context, path string, initialize func(context.Context, *sql.DB) error) (*sql.DB, error) {
	db, err := sql.Open(sqliteDriverName, sqliteDSN(path))
	if err != nil {
		return nil, fmt.Errorf("store: open sqlite database %q: %w", path, err)
	}

	db.SetMaxOpenConns(defaultMaxOpenConns)
	db.SetMaxIdleConns(defaultMaxIdleConns)

	if err := db.PingContext(ctx); err != nil {
		closeQuietly(db)
		return nil, fmt.Errorf("store: ping sqlite database %q: %w", path, err)
	}
	if err := configureSQLite(ctx, db); err != nil {
		closeQuietly(db)
		return nil, fmt.Errorf("store: configure sqlite database %q: %w", path, err)
	}
	if initialize != nil {
		if err := initialize(ctx, db); err != nil {
			closeQuietly(db)
			return nil, fmt.Errorf("store: initialize sqlite database %q: %w", path, err)
		}
	}

	return db, nil
}

func sqliteDSN(path string) string {
	u := url.URL{
		Scheme: "file",
		Path:   filepath.ToSlash(path),
	}
	return u.String()
}

func configureSQLite(ctx context.Context, db *sql.DB) error {
	if _, err := db.ExecContext(ctx, fmt.Sprintf("PRAGMA busy_timeout = %d", defaultBusyTimeoutMS)); err != nil {
		return err
	}

	mode, err := querySingleString(ctx, db, "PRAGMA journal_mode = WAL")
	if err != nil {
		return err
	}
	if !strings.EqualFold(mode, "wal") {
		return fmt.Errorf("store: sqlite journal_mode = %q, want wal", mode)
	}

	if _, err := db.ExecContext(ctx, "PRAGMA synchronous = NORMAL"); err != nil {
		return err
	}
	if _, err := db.ExecContext(ctx, "PRAGMA foreign_keys = ON"); err != nil {
		return err
	}

	return nil
}

func querySingleString(ctx context.Context, db *sql.DB, stmt string) (string, error) {
	var value string
	if err := db.QueryRowContext(ctx, stmt).Scan(&value); err != nil {
		return "", err
	}
	return value, nil
}

func ensureSchema(ctx context.Context, db *sql.DB, statements []string) error {
	for _, stmt := range statements {
		if _, err := db.ExecContext(ctx, stmt); err != nil {
			return err
		}
	}
	return nil
}

type sqlQueryExecutor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

type legacySessionRow struct {
	ID           string
	Name         sql.NullString
	AgentName    string
	Workspace    string
	SessionType  string
	State        string
	ACPSessionID sql.NullString
	CreatedAt    string
	UpdatedAt    string
}

type legacyWorkspaceSeed struct {
	rootDir   string
	createdAt string
	updatedAt string
}

func migrateGlobalSchema(ctx context.Context, db *sql.DB) error {
	hasSessions, err := tableExists(ctx, db, "sessions")
	if err != nil {
		return err
	}
	if !hasSessions {
		return nil
	}

	columns, err := tableColumns(ctx, db, "sessions")
	if err != nil {
		return err
	}
	if _, ok := columns["workspace_id"]; ok {
		return nil
	}
	if _, ok := columns["workspace"]; !ok {
		return nil
	}

	if _, err := db.ExecContext(ctx, "PRAGMA foreign_keys = OFF"); err != nil {
		return fmt.Errorf("store: disable foreign keys for global schema migration: %w", err)
	}
	defer func() {
		_, _ = db.ExecContext(context.Background(), "PRAGMA foreign_keys = ON")
	}()

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("store: begin global schema migration transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	if _, err := tx.ExecContext(ctx, globalSchemaStatements[0]); err != nil {
		return fmt.Errorf("store: create workspaces table during migration: %w", err)
	}

	sessionRows, workspaceSeeds, err := loadLegacySessions(ctx, tx)
	if err != nil {
		return err
	}

	workspaceIDs, err := ensureMigratedWorkspaces(ctx, tx, workspaceSeeds)
	if err != nil {
		return err
	}

	if err := createMigratedGlobalTables(ctx, tx); err != nil {
		return err
	}
	if err := copyMigratedSessions(ctx, tx, sessionRows, workspaceIDs); err != nil {
		return err
	}
	if err := copyGlobalTableIfExists(ctx, tx, "event_summaries", "event_summaries_new", `INSERT INTO event_summaries_new (id, session_id, type, agent_name, summary, timestamp) SELECT id, session_id, type, agent_name, summary, timestamp FROM event_summaries`); err != nil {
		return err
	}
	if err := copyGlobalTableIfExists(ctx, tx, "token_stats", "token_stats_new", `INSERT INTO token_stats_new (id, session_id, agent_name, input_tokens, output_tokens, total_tokens, total_cost, cost_currency, turn_count, updated_at) SELECT id, session_id, agent_name, input_tokens, output_tokens, total_tokens, total_cost, cost_currency, turn_count, updated_at FROM token_stats`); err != nil {
		return err
	}
	if err := copyGlobalTableIfExists(ctx, tx, "permission_log", "permission_log_new", `INSERT INTO permission_log_new (id, session_id, agent_name, action, resource, decision, policy_used, timestamp) SELECT id, session_id, agent_name, action, resource, decision, policy_used, timestamp FROM permission_log`); err != nil {
		return err
	}
	if err := swapMigratedGlobalTables(ctx, tx); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("store: commit global schema migration: %w", err)
	}

	return nil
}

func loadLegacySessions(ctx context.Context, exec sqlQueryExecutor) ([]legacySessionRow, map[string]legacyWorkspaceSeed, error) {
	rows, err := exec.QueryContext(ctx, `SELECT id, name, agent_name, workspace, session_type, state, acp_session_id, created_at, updated_at FROM sessions ORDER BY created_at ASC, id ASC`)
	if err != nil {
		return nil, nil, fmt.Errorf("store: query legacy sessions for migration: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	sessions := make([]legacySessionRow, 0)
	seeds := make(map[string]legacyWorkspaceSeed)
	for rows.Next() {
		var row legacySessionRow
		if err := rows.Scan(
			&row.ID,
			&row.Name,
			&row.AgentName,
			&row.Workspace,
			&row.SessionType,
			&row.State,
			&row.ACPSessionID,
			&row.CreatedAt,
			&row.UpdatedAt,
		); err != nil {
			return nil, nil, fmt.Errorf("store: scan legacy session for migration: %w", err)
		}

		rootDir := strings.TrimSpace(row.Workspace)
		if rootDir == "" {
			return nil, nil, fmt.Errorf("store: migrate legacy session %q: workspace path is required", row.ID)
		}

		seed, ok := seeds[rootDir]
		if !ok {
			seeds[rootDir] = legacyWorkspaceSeed{rootDir: rootDir, createdAt: row.CreatedAt, updatedAt: row.UpdatedAt}
		} else {
			if strings.TrimSpace(row.CreatedAt) != "" && (seed.createdAt == "" || row.CreatedAt < seed.createdAt) {
				seed.createdAt = row.CreatedAt
			}
			if strings.TrimSpace(row.UpdatedAt) != "" && row.UpdatedAt > seed.updatedAt {
				seed.updatedAt = row.UpdatedAt
			}
			seeds[rootDir] = seed
		}

		sessions = append(sessions, row)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("store: iterate legacy sessions for migration: %w", err)
	}

	return sessions, seeds, nil
}

func ensureMigratedWorkspaces(ctx context.Context, tx *sql.Tx, seeds map[string]legacyWorkspaceSeed) (map[string]string, error) {
	rootToID, err := loadWorkspaceIDsByRootDir(ctx, tx)
	if err != nil {
		return nil, err
	}
	takenNames, err := loadWorkspaceNames(ctx, tx)
	if err != nil {
		return nil, err
	}

	if len(seeds) == 0 {
		return rootToID, nil
	}

	orderedRoots := make([]string, 0, len(seeds))
	for rootDir := range seeds {
		orderedRoots = append(orderedRoots, rootDir)
	}
	sort.Strings(orderedRoots)

	for _, rootDir := range orderedRoots {
		if _, ok := rootToID[rootDir]; ok {
			continue
		}

		seed := seeds[rootDir]
		name := uniqueWorkspaceName(rootDir, takenNames)
		workspaceID := newID("ws")
		if _, err := tx.ExecContext(
			ctx,
			`INSERT INTO workspaces (id, root_dir, add_dirs, name, default_agent, created_at, updated_at)
			 VALUES (?, ?, '[]', ?, '', ?, ?)`,
			workspaceID,
			rootDir,
			name,
			coalesceTimestamp(seed.createdAt),
			coalesceTimestamp(seed.updatedAt),
		); err != nil {
			return nil, fmt.Errorf("store: insert migrated workspace for %q: %w", rootDir, err)
		}

		rootToID[rootDir] = workspaceID
		takenNames[name] = struct{}{}
	}

	return rootToID, nil
}

func createMigratedGlobalTables(ctx context.Context, tx *sql.Tx) error {
	statements := []string{
		`CREATE TABLE sessions_new (
			id             TEXT PRIMARY KEY,
			name           TEXT,
			agent_name     TEXT NOT NULL,
			workspace_id   TEXT NOT NULL REFERENCES workspaces(id),
			session_type   TEXT NOT NULL DEFAULT 'user',
			state          TEXT NOT NULL,
			acp_session_id TEXT,
			created_at     TEXT NOT NULL,
			updated_at     TEXT NOT NULL
		);`,
		`CREATE TABLE event_summaries_new (
			id         TEXT PRIMARY KEY,
			session_id TEXT NOT NULL REFERENCES sessions(id),
			type       TEXT NOT NULL,
			agent_name TEXT NOT NULL,
			summary    TEXT,
			timestamp  TEXT NOT NULL
		);`,
		`CREATE TABLE token_stats_new (
			id            TEXT PRIMARY KEY,
			session_id    TEXT NOT NULL REFERENCES sessions(id),
			agent_name    TEXT NOT NULL,
			input_tokens  INTEGER,
			output_tokens INTEGER,
			total_tokens  INTEGER,
			total_cost    REAL,
			cost_currency TEXT,
			turn_count    INTEGER NOT NULL DEFAULT 0,
			updated_at    TEXT NOT NULL
		);`,
		`CREATE TABLE permission_log_new (
			id          TEXT PRIMARY KEY,
			session_id  TEXT NOT NULL REFERENCES sessions(id),
			agent_name  TEXT NOT NULL,
			action      TEXT NOT NULL,
			resource    TEXT NOT NULL,
			decision    TEXT NOT NULL,
			policy_used TEXT NOT NULL,
			timestamp   TEXT NOT NULL
		);`,
	}

	for _, stmt := range statements {
		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("store: create migrated global table: %w", err)
		}
	}

	return nil
}

func copyMigratedSessions(ctx context.Context, tx *sql.Tx, sessions []legacySessionRow, workspaceIDs map[string]string) error {
	for _, row := range sessions {
		workspaceID, ok := workspaceIDs[strings.TrimSpace(row.Workspace)]
		if !ok {
			return fmt.Errorf("store: missing migrated workspace id for legacy root %q", row.Workspace)
		}

		if _, err := tx.ExecContext(
			ctx,
			`INSERT INTO sessions_new (
				id, name, agent_name, workspace_id, session_type, state, acp_session_id, created_at, updated_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			row.ID,
			nullStringValue(row.Name),
			row.AgentName,
			workspaceID,
			normalizeSessionType(row.SessionType),
			row.State,
			nullStringValue(row.ACPSessionID),
			row.CreatedAt,
			row.UpdatedAt,
		); err != nil {
			return fmt.Errorf("store: copy migrated session %q: %w", row.ID, err)
		}
	}

	return nil
}

func copyGlobalTableIfExists(ctx context.Context, tx *sql.Tx, source string, target string, insertSQL string) error {
	exists, err := tableExists(ctx, tx, source)
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}
	if _, err := tx.ExecContext(ctx, insertSQL); err != nil {
		return fmt.Errorf("store: copy %s into %s: %w", source, target, err)
	}
	return nil
}

func swapMigratedGlobalTables(ctx context.Context, tx *sql.Tx) error {
	statements := []string{
		`DROP TABLE IF EXISTS event_summaries`,
		`DROP TABLE IF EXISTS token_stats`,
		`DROP TABLE IF EXISTS permission_log`,
		`DROP TABLE IF EXISTS sessions`,
		`ALTER TABLE sessions_new RENAME TO sessions`,
		`ALTER TABLE event_summaries_new RENAME TO event_summaries`,
		`ALTER TABLE token_stats_new RENAME TO token_stats`,
		`ALTER TABLE permission_log_new RENAME TO permission_log`,
	}

	for _, stmt := range statements {
		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("store: swap migrated global tables: %w", err)
		}
	}

	return nil
}

func loadWorkspaceIDsByRootDir(ctx context.Context, exec sqlQueryExecutor) (map[string]string, error) {
	exists, err := tableExists(ctx, exec, "workspaces")
	if err != nil {
		return nil, err
	}
	if !exists {
		return map[string]string{}, nil
	}

	rows, err := exec.QueryContext(ctx, `SELECT id, root_dir FROM workspaces`)
	if err != nil {
		return nil, fmt.Errorf("store: query workspace ids by root_dir: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	rootToID := make(map[string]string)
	for rows.Next() {
		var id string
		var rootDir string
		if err := rows.Scan(&id, &rootDir); err != nil {
			return nil, fmt.Errorf("store: scan workspace id by root_dir: %w", err)
		}
		rootToID[strings.TrimSpace(rootDir)] = strings.TrimSpace(id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: iterate workspace ids by root_dir: %w", err)
	}

	return rootToID, nil
}

func loadWorkspaceNames(ctx context.Context, exec sqlQueryExecutor) (map[string]struct{}, error) {
	exists, err := tableExists(ctx, exec, "workspaces")
	if err != nil {
		return nil, err
	}
	if !exists {
		return map[string]struct{}{}, nil
	}

	rows, err := exec.QueryContext(ctx, `SELECT name FROM workspaces`)
	if err != nil {
		return nil, fmt.Errorf("store: query workspace names: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	names := make(map[string]struct{})
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("store: scan workspace name: %w", err)
		}
		names[strings.TrimSpace(name)] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: iterate workspace names: %w", err)
	}

	return names, nil
}

func tableExists(ctx context.Context, exec sqlQueryExecutor, table string) (bool, error) {
	var name string
	err := exec.QueryRowContext(ctx, `SELECT name FROM sqlite_master WHERE type = 'table' AND name = ?`, strings.TrimSpace(table)).Scan(&name)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("store: check table %q existence: %w", table, err)
	}
	return true, nil
}

func tableColumns(ctx context.Context, exec sqlQueryExecutor, table string) (map[string]struct{}, error) {
	rows, err := exec.QueryContext(ctx, fmt.Sprintf("PRAGMA table_info(%s)", strings.TrimSpace(table)))
	if err != nil {
		return nil, fmt.Errorf("store: query table info for %q: %w", table, err)
	}
	defer func() {
		_ = rows.Close()
	}()

	columns := make(map[string]struct{})
	for rows.Next() {
		var (
			cid        int
			name       string
			columnType string
			notNull    int
			defaultVal sql.NullString
			primaryKey int
		)
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultVal, &primaryKey); err != nil {
			return nil, fmt.Errorf("store: scan table info for %q: %w", table, err)
		}
		columns[strings.TrimSpace(name)] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: iterate table info for %q: %w", table, err)
	}

	return columns, nil
}

func uniqueWorkspaceName(rootDir string, taken map[string]struct{}) string {
	baseName := filepath.Base(filepath.Clean(strings.TrimSpace(rootDir)))
	switch baseName {
	case "", ".", string(filepath.Separator):
		baseName = "workspace"
	}

	candidate := baseName
	for suffix := 2; ; suffix++ {
		if _, ok := taken[candidate]; !ok {
			return candidate
		}
		candidate = fmt.Sprintf("%s-%d", baseName, suffix)
	}
}

func coalesceTimestamp(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed != "" {
		return trimmed
	}
	return formatTimestamp(time.Now().UTC())
}

func nullStringValue(value sql.NullString) any {
	if !value.Valid {
		return nil
	}
	trimmed := strings.TrimSpace(value.String)
	if trimmed == "" {
		return nil
	}
	return trimmed
}

func checkpoint(ctx context.Context, db *sql.DB) error {
	if db == nil {
		return nil
	}
	if _, err := db.ExecContext(ctx, "PRAGMA wal_checkpoint(TRUNCATE)"); err != nil {
		return fmt.Errorf("store: checkpoint sqlite wal: %w", err)
	}
	return nil
}

func recoverSQLiteDatabase(path string) (string, error) {
	corruptPath := fmt.Sprintf("%s.corrupt.%s", path, time.Now().UTC().Format("20060102T150405.000000000Z0700"))
	if err := os.Rename(path, corruptPath); err != nil {
		return "", err
	}
	return corruptPath, nil
}

func shouldRecoverSQLite(err error) bool {
	if err == nil {
		return false
	}

	message := strings.ToLower(err.Error())
	for _, marker := range []string{
		"not a database",
		"database disk image is malformed",
		"malformed database schema",
		"malformed",
		"file is encrypted or is not a database",
	} {
		if strings.Contains(message, marker) {
			return true
		}
	}

	return false
}

func closeQuietly(db *sql.DB) {
	if db != nil {
		_ = db.Close()
	}
}
