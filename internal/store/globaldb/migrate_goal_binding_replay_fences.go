package globaldb

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/compozy/agh/internal/store"
)

// Migration 80: fence borrowed-origin re-adoption and binding-retry replay.
//
// Why: both transitions need immutable source-owner evidence after their source row changes.
// Affects: loop_session_bindings and new table loop_goal_binding_retry_witnesses.
// Idempotent: yes; the guarded column and IF NOT EXISTS table tolerate a registry retry.
// Reversible: no; both values are durable lifecycle evidence.
var goalBindingReplayFencesMigration = store.Migration{
	Version:  80,
	Name:     "add_goal_binding_replay_fences",
	Up:       migrateGoalBindingReplayFences,
	Checksum: "2026-07-11-add-goal-binding-replay-fences",
}

func migrateGoalBindingReplayFences(ctx context.Context, tx *sql.Tx) error {
	if err := addMissingMigrationColumns(ctx, tx, "loop_session_bindings", []migrationColumnSpec{
		{
			name: "adopted_generation",
			sql: `ALTER TABLE loop_session_bindings ADD COLUMN adopted_generation INTEGER
				NOT NULL DEFAULT 0 CHECK (adopted_generation >= 0)`,
		},
	}); err != nil {
		return fmt.Errorf("store: add Goal binding adoption generation: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS loop_goal_binding_retry_witnesses (
		loop_run_id          TEXT NOT NULL REFERENCES loop_runs(id) ON DELETE CASCADE,
		handle               TEXT NOT NULL CHECK (length(trim(handle)) > 0),
		failed_binding_epoch INTEGER NOT NULL CHECK (failed_binding_epoch >= 1),
		request_digest       TEXT NOT NULL CHECK (length(request_digest) = 64),
		created_at           TIMESTAMP NOT NULL,
		PRIMARY KEY (loop_run_id, handle, failed_binding_epoch)
	)`); err != nil {
		return fmt.Errorf("store: create Goal binding retry witness schema: %w", err)
	}
	return nil
}
