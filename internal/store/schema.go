package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
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
	`CREATE TABLE IF NOT EXISTS sessions (
		id             TEXT PRIMARY KEY,
		name           TEXT,
		agent_name     TEXT NOT NULL,
		workspace      TEXT NOT NULL,
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
	return openSQLiteDatabase(ctx, path, sessionSchemaStatements)
}

func openGlobalSQLite(ctx context.Context, path string) (*sql.DB, error) {
	return openSQLiteDatabase(ctx, path, globalSchemaStatements)
}

func openSQLiteDatabase(ctx context.Context, path string, schema []string) (*sql.DB, error) {
	cleanPath := strings.TrimSpace(path)
	if cleanPath == "" {
		return nil, errors.New("store: database path is required")
	}
	if err := os.MkdirAll(filepath.Dir(cleanPath), 0o755); err != nil {
		return nil, fmt.Errorf("store: create database directory for %q: %w", cleanPath, err)
	}

	db, err := openSQLiteDatabaseOnce(ctx, cleanPath, schema)
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

	db, reopenErr := openSQLiteDatabaseOnce(ctx, cleanPath, schema)
	if reopenErr != nil {
		return nil, errors.Join(err, fmt.Errorf("store: reopen sqlite database %q after recovery: %w", cleanPath, reopenErr))
	}
	return db, nil
}

func openSQLiteDatabaseOnce(ctx context.Context, path string, schema []string) (*sql.DB, error) {
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
	if err := ensureSchema(ctx, db, schema); err != nil {
		closeQuietly(db)
		return nil, fmt.Errorf("store: ensure sqlite schema for %q: %w", path, err)
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
