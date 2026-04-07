package store

import (
	"context"
	"database/sql"
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
		if err := ensureSchema(ctx, db, globalSchemaStatements); err != nil {
			return err
		}
		return reconcileLegacySessionMetaWorkspaceIDs(ctx, db, sessionsDirForDatabasePath(path))
	})
}

func ensureSchema(ctx context.Context, db *sql.DB, statements []string) error {
	for _, stmt := range statements {
		if _, err := db.ExecContext(ctx, stmt); err != nil {
			return err
		}
	}
	return nil
}
