package globaldb

import (
	"context"
	"database/sql"
	"fmt"
)

func migrateLoopRunIterationCap(ctx context.Context, tx *sql.Tx) error {
	if err := addMissingMigrationColumns(ctx, tx, "loop_runs", []migrationColumnSpec{
		{
			name: "iteration_cap",
			sql:  "ALTER TABLE loop_runs ADD COLUMN iteration_cap INTEGER NOT NULL DEFAULT 0",
		},
	}); err != nil {
		return fmt.Errorf("store: add loop_runs.iteration_cap column: %w", err)
	}
	return nil
}
