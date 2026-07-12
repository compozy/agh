package globaldb

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/compozy/agh/internal/store"
)

// Migration 79: persist Goal compaction comparison state and run-owned session cleanup obligations.
//
// Why: delayed context telemetry and session Stop effects must remain crash-equivalent across daemon restart.
// Affects: agh.db table loop_goal_checkpoints and new table loop_goal_session_cleanup.
// Idempotent: yes; guarded columns and IF NOT EXISTS table/index tolerate a registry retry.
// Reversible: no; both records are durable recovery evidence.
var goalRecoveryCleanupMigration = store.Migration{
	Version:  79,
	Name:     "add_goal_recovery_cleanup",
	Up:       migrateGoalRecoveryCleanup,
	Checksum: "2026-07-11-add-goal-recovery-cleanup",
}

func migrateGoalRecoveryCleanup(ctx context.Context, tx *sql.Tx) error {
	if err := addMissingMigrationColumns(ctx, tx, "loop_goal_checkpoints", []migrationColumnSpec{
		{
			name: "compaction_baseline_used",
			sql: `ALTER TABLE loop_goal_checkpoints ADD COLUMN compaction_baseline_used INTEGER
				CHECK (compaction_baseline_used IS NULL OR compaction_baseline_used >= 0)`,
		},
		{
			name: "compaction_recovery_required",
			sql: `ALTER TABLE loop_goal_checkpoints ADD COLUMN compaction_recovery_required INTEGER
				NOT NULL DEFAULT 0 CHECK (compaction_recovery_required IN (0,1))`,
		},
	}); err != nil {
		return fmt.Errorf("store: add Goal compaction recovery state: %w", err)
	}
	for _, statement := range []string{
		`CREATE TABLE IF NOT EXISTS loop_goal_session_cleanup (
			id            INTEGER PRIMARY KEY AUTOINCREMENT,
			cleanup_id    TEXT NOT NULL UNIQUE CHECK (length(trim(cleanup_id)) > 0),
			workspace_id  TEXT NOT NULL CHECK (length(trim(workspace_id)) > 0),
			loop_run_id   TEXT NOT NULL REFERENCES loop_runs(id) ON DELETE CASCADE,
			handle        TEXT NOT NULL CHECK (length(trim(handle)) > 0),
			binding_epoch INTEGER NOT NULL CHECK (binding_epoch >= 1),
			session_id    TEXT NOT NULL CHECK (length(trim(session_id)) > 0),
			cause         TEXT NOT NULL CHECK (cause IN ('terminal','reseed','control-revoked','stop')),
			created_at    TIMESTAMP NOT NULL,
			completed_at  TIMESTAMP CHECK (completed_at IS NULL OR completed_at >= created_at),
			UNIQUE (loop_run_id, handle, binding_epoch)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_loop_goal_session_cleanup_pending
			ON loop_goal_session_cleanup(id) WHERE completed_at IS NULL`,
	} {
		if _, err := tx.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("store: create Goal session cleanup schema: %w", err)
		}
	}
	return nil
}
