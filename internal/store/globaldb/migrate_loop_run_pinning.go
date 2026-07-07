package globaldb

import (
	"context"
	"database/sql"
	"fmt"
)

const loopDefinitionSnapshotsTableStatement = `CREATE TABLE IF NOT EXISTS loop_definition_snapshots (
			workspace_id        TEXT NOT NULL,
			definition_digest  TEXT NOT NULL,
			definition_version INTEGER NOT NULL DEFAULT 0,
			definition_json    TEXT NOT NULL,
			byte_size          INTEGER NOT NULL CHECK (byte_size >= 0),
			created_at         TEXT NOT NULL,
			last_used_at       TEXT NOT NULL,
			PRIMARY KEY (workspace_id, definition_digest)
		);`

const loopGateDecisionsTableStatement = `CREATE TABLE IF NOT EXISTS loop_gate_decisions (
			workspace_id  TEXT NOT NULL,
			loop_run_id   TEXT NOT NULL REFERENCES loop_runs(id) ON DELETE CASCADE,
			generation    INTEGER NOT NULL,
			gate_id       TEXT NOT NULL,
			criterion_id  TEXT NOT NULL,
			decision      TEXT NOT NULL,
			actor_kind    TEXT NOT NULL,
			actor_ref     TEXT NOT NULL,
			origin_kind   TEXT NOT NULL,
			origin_ref    TEXT NOT NULL,
			note          TEXT NOT NULL DEFAULT '',
			decided_at    TEXT NOT NULL,
			PRIMARY KEY (loop_run_id, generation, gate_id, criterion_id)
		);`

const loopGateDecisionsWorkspaceIndexStatement = `CREATE INDEX IF NOT EXISTS idx_loop_gate_decisions_workspace_run
			ON loop_gate_decisions(workspace_id, loop_run_id, generation, gate_id);`

func migrateLoopRunPinning(ctx context.Context, tx *sql.Tx) error {
	const (
		bootstrapStartedAt     = "1970-01-01T00:00:00.000000000Z"
		loopRunStartedAtColumn = "started_at"
	)
	if err := addMissingMigrationColumns(ctx, tx, "loop_runs", []migrationColumnSpec{
		{
			name: loopRunStartedAtColumn,
			sql:  `ALTER TABLE loop_runs ADD COLUMN started_at TEXT NOT NULL DEFAULT '` + bootstrapStartedAt + `'`,
		},
		{
			name: "definition_version",
			sql:  "ALTER TABLE loop_runs ADD COLUMN definition_version INTEGER NOT NULL DEFAULT 0",
		},
		{
			name: "definition_digest",
			sql:  "ALTER TABLE loop_runs ADD COLUMN definition_digest TEXT NOT NULL DEFAULT ''",
		},
		{
			name: "active_gate_id",
			sql:  "ALTER TABLE loop_runs ADD COLUMN active_gate_id TEXT NOT NULL DEFAULT ''",
		},
		{
			name: "active_human_criteria_json",
			sql:  "ALTER TABLE loop_runs ADD COLUMN active_human_criteria_json TEXT NOT NULL DEFAULT '[]'",
		},
		{
			name: "budget_approval_seq",
			sql:  "ALTER TABLE loop_runs ADD COLUMN budget_approval_seq INTEGER NOT NULL DEFAULT 0",
		},
		{
			name: "start_metadata_json",
			sql:  "ALTER TABLE loop_runs ADD COLUMN start_metadata_json TEXT NOT NULL DEFAULT '{}'",
		},
	}); err != nil {
		return fmt.Errorf("store: add loop run pinning columns: %w", err)
	}
	if _, err := tx.ExecContext(
		ctx,
		`UPDATE loop_runs
		 SET started_at = created_at
		 WHERE started_at = ?`,
		bootstrapStartedAt,
	); err != nil {
		return fmt.Errorf("store: backfill loop_runs.started_at: %w", err)
	}
	if _, err := tx.ExecContext(ctx, loopDefinitionSnapshotsTableStatement); err != nil {
		return fmt.Errorf("store: create loop definition snapshots table: %w", err)
	}
	if _, err := tx.ExecContext(ctx, loopGateDecisionsTableStatement); err != nil {
		return fmt.Errorf("store: create loop gate decisions table: %w", err)
	}
	if _, err := tx.ExecContext(ctx, loopGateDecisionsWorkspaceIndexStatement); err != nil {
		return fmt.Errorf("store: create loop gate decisions workspace index: %w", err)
	}
	return nil
}
