package globaldb

import (
	"context"
	"database/sql"
	"fmt"
)

func migrateLoopRunStartActor(ctx context.Context, tx *sql.Tx) error {
	if err := addMissingMigrationColumns(ctx, tx, "loop_runs", []migrationColumnSpec{
		{
			name: "started_by_kind",
			sql:  "ALTER TABLE loop_runs ADD COLUMN started_by_kind TEXT NOT NULL DEFAULT ''",
		},
		{
			name: "started_by_ref",
			sql:  "ALTER TABLE loop_runs ADD COLUMN started_by_ref TEXT NOT NULL DEFAULT ''",
		},
		{
			name: "started_origin_kind",
			sql:  "ALTER TABLE loop_runs ADD COLUMN started_origin_kind TEXT NOT NULL DEFAULT ''",
		},
		{
			name: "started_origin_ref",
			sql:  "ALTER TABLE loop_runs ADD COLUMN started_origin_ref TEXT NOT NULL DEFAULT ''",
		},
	}); err != nil {
		return fmt.Errorf("store: add loop_runs start actor columns: %w", err)
	}
	return nil
}
