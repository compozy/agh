package globaldb

import (
	"context"
	"database/sql"
	"fmt"
)

func migrateSessionsWorkspaceStateIndex(ctx context.Context, tx *sql.Tx) error {
	exists, err := tableExists(ctx, tx, "sessions")
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}
	if _, err := tx.ExecContext(
		ctx,
		`CREATE INDEX IF NOT EXISTS idx_sessions_workspace_state ON sessions(workspace_id, state)`,
	); err != nil {
		return fmt.Errorf("store: add sessions workspace/state index: %w", err)
	}
	return nil
}
