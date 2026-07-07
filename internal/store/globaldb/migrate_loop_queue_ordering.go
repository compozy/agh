package globaldb

import (
	"context"
	"database/sql"
	"fmt"
)

const loopRunQueueOrderingIndexStatement = `CREATE INDEX IF NOT EXISTS idx_loop_runs_queue_order
			ON loop_runs(workspace_id, loop_name, status, created_at ASC, id ASC);`

const loopRunsCreatedAtColumn = "created_at"

func migrateLoopRunQueueOrdering(ctx context.Context, tx *sql.Tx) error {
	const bootstrapCreatedAt = "1970-01-01T00:00:00.000000000Z"
	// Migration 51 is an append-only follow-on to migration 50: created_at is required for
	// FIFO queue promotion once loop_runs already exists in local alpha databases.
	if err := addMissingMigrationColumns(ctx, tx, "loop_runs", []migrationColumnSpec{
		{
			name: loopRunsCreatedAtColumn,
			sql:  `ALTER TABLE loop_runs ADD COLUMN created_at TEXT NOT NULL DEFAULT '` + bootstrapCreatedAt + `'`,
		},
	}); err != nil {
		return err
	}
	if _, err := tx.ExecContext(
		ctx,
		`UPDATE loop_runs
		 SET created_at = last_progress_at
		 WHERE created_at = ?`,
		bootstrapCreatedAt,
	); err != nil {
		return fmt.Errorf("store: backfill loop_runs.created_at: %w", err)
	}
	if _, err := tx.ExecContext(ctx, loopRunQueueOrderingIndexStatement); err != nil {
		return fmt.Errorf("store: create loop run queue ordering index: %w", err)
	}
	return nil
}
