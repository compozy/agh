package globaldb

import (
	"context"
	"database/sql"
	"fmt"
)

func migrateLoopRunStateSchema(ctx context.Context, conn *sql.Conn) (err error) {
	if _, err := conn.ExecContext(ctx, "BEGIN IMMEDIATE"); err != nil {
		return fmt.Errorf("store: begin loop run state schema migration: %w", err)
	}
	rollbackCtx, rollbackCancel := notificationCursorRollbackContext(ctx)
	defer rollbackCancel()
	finished := false
	defer func() {
		if !finished {
			joinCleanupError(&err, rollbackImmediate(rollbackCtx, conn, "loop run state schema migration"))
		}
	}()
	if err := addMissingMigrationColumns(ctx, conn, "task_runs", []migrationColumnSpec{
		{
			name: "run_kind",
			sql:  `ALTER TABLE task_runs ADD COLUMN run_kind TEXT NOT NULL DEFAULT 'worker'`,
		},
		{
			name: columnLoopRunID,
			sql:  `ALTER TABLE task_runs ADD COLUMN loop_run_id TEXT`,
		},
		{
			name: columnTokensUsed,
			sql:  `ALTER TABLE task_runs ADD COLUMN tokens_used INTEGER NOT NULL DEFAULT 0`,
		},
	}); err != nil {
		return fmt.Errorf("store: add loop run state migration columns: %w", err)
	}
	for _, statement := range loopRunStateSchemaStatements() {
		if _, err := conn.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("store: apply loop run state schema: %w", err)
		}
	}
	if _, err := conn.ExecContext(ctx, "COMMIT"); err != nil {
		return fmt.Errorf("store: commit loop run state schema migration: %w", err)
	}
	finished = true
	return nil
}
