package globaldb

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/compozy/agh/internal/store"
)

// Migration 76: persist the winning Goal checkpoint control cause.
//
// Why: a reclaimed worker must reproduce the exact typed control without inferring from prose or events.
// Affects: agh.db table loop_goal_checkpoints.
// Idempotent: yes; the guarded column addition tolerates a registry retry.
// Reversible: no; the cause is durable recovery evidence.
var goalCheckpointControlCauseMigration = store.Migration{
	Version:  76,
	Name:     "add_goal_checkpoint_control_cause",
	Up:       migrateGoalCheckpointControlCause,
	Checksum: "2026-07-10-add-goal-checkpoint-control-cause",
}

func migrateGoalCheckpointControlCause(ctx context.Context, tx *sql.Tx) error {
	if err := addMissingMigrationColumns(ctx, tx, "loop_goal_checkpoints", []migrationColumnSpec{
		{name: "control_cause", sql: `ALTER TABLE loop_goal_checkpoints ADD COLUMN control_cause TEXT`},
	}); err != nil {
		return fmt.Errorf("store: add Goal checkpoint control cause: %w", err)
	}
	return nil
}
