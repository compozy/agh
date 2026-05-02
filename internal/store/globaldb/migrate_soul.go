package globaldb

import (
	"context"
	"database/sql"
	"fmt"
)

func migrateAgentSoulSnapshots(ctx context.Context, tx *sql.Tx) error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS agent_soul_snapshots (
			id            TEXT PRIMARY KEY,
			workspace_id  TEXT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
			agent_name    TEXT NOT NULL,
			source_path   TEXT NOT NULL,
			digest        TEXT NOT NULL,
			profile_json  TEXT NOT NULL DEFAULT '{}',
			body          TEXT NOT NULL DEFAULT '',
			truncated     INTEGER NOT NULL DEFAULT 0 CHECK (truncated IN (0, 1)),
			created_at    TEXT NOT NULL,
			UNIQUE (workspace_id, agent_name, digest)
		);`,
		`CREATE INDEX IF NOT EXISTS idx_agent_soul_snapshots_agent
			ON agent_soul_snapshots(workspace_id, agent_name, created_at DESC);`,
		`CREATE TABLE IF NOT EXISTS agent_soul_revisions (
			id               TEXT PRIMARY KEY,
			workspace_id     TEXT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
			agent_name       TEXT NOT NULL,
			source_path      TEXT NOT NULL,
			action           TEXT NOT NULL CHECK (action IN ('put', 'delete', 'rollback')),
			previous_digest  TEXT NOT NULL DEFAULT '',
			new_digest       TEXT NOT NULL DEFAULT '',
			body             TEXT NOT NULL DEFAULT '',
			diagnostics_json TEXT NOT NULL DEFAULT '[]',
			actor_kind       TEXT NOT NULL DEFAULT '',
			actor_ref        TEXT NOT NULL DEFAULT '',
			origin_kind      TEXT NOT NULL DEFAULT '',
			origin_ref       TEXT NOT NULL DEFAULT '',
			created_at       TEXT NOT NULL
		);`,
		`CREATE INDEX IF NOT EXISTS idx_agent_soul_revisions_agent
			ON agent_soul_revisions(workspace_id, agent_name, created_at DESC);`,
	}
	for _, statement := range statements {
		if _, err := tx.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("store: migrate agent soul tables: %w", err)
		}
	}

	columns, err := tableColumns(ctx, tx, "sessions")
	if err != nil {
		return err
	}
	sessionColumns := []struct {
		name string
		sql  string
	}{
		{
			name: "soul_snapshot_id",
			sql: `ALTER TABLE sessions ADD COLUMN soul_snapshot_id TEXT
				REFERENCES agent_soul_snapshots(id) ON DELETE SET NULL`,
		},
		{name: "soul_digest", sql: `ALTER TABLE sessions ADD COLUMN soul_digest TEXT NOT NULL DEFAULT ''`},
		{
			name: "parent_soul_digest",
			sql:  `ALTER TABLE sessions ADD COLUMN parent_soul_digest TEXT NOT NULL DEFAULT ''`,
		},
	}
	for _, column := range sessionColumns {
		if _, ok := columns[column.name]; ok {
			continue
		}
		if _, err := tx.ExecContext(ctx, column.sql); err != nil {
			return fmt.Errorf("store: add sessions.%s column: %w", column.name, err)
		}
	}
	if _, err := tx.ExecContext(
		ctx,
		`CREATE INDEX IF NOT EXISTS idx_sessions_soul_snapshot
			ON sessions(soul_snapshot_id);`,
	); err != nil {
		return fmt.Errorf("store: create sessions soul snapshot index: %w", err)
	}
	return nil
}
