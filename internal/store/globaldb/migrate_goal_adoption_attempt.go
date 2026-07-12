package globaldb

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/compozy/agh/internal/store"
)

// Migration 81: persist the generation-scoped borrowed-origin adoption attempt.
var goalAdoptionAttemptMigration = store.Migration{
	Version:  81,
	Name:     "add_goal_adoption_attempt",
	Up:       migrateGoalAdoptionAttempt,
	Checksum: "2026-07-11-add-goal-adoption-attempt",
}

func migrateGoalAdoptionAttempt(ctx context.Context, tx *sql.Tx) error {
	if err := addMissingMigrationColumns(ctx, tx, "loop_session_bindings", []migrationColumnSpec{{
		name: "adoption_attempt_id",
		sql: `ALTER TABLE loop_session_bindings ADD COLUMN adoption_attempt_id TEXT
			CHECK (adoption_attempt_id IS NULL OR length(trim(adoption_attempt_id)) > 0)`,
	}}); err != nil {
		return fmt.Errorf("store: add Goal adoption attempt: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `UPDATE loop_session_bindings
		SET adoption_attempt_id = binding_attempt_id
		WHERE ownership = 'origin-borrowed' AND adoption_attempt_id IS NULL`); err != nil {
		return fmt.Errorf("store: backfill Goal adoption attempt: %w", err)
	}
	return nil
}
