package globaldb

import (
	"context"
	"database/sql"
	"fmt"
)

func migrateLoopOutputBlobs(ctx context.Context, tx *sql.Tx) error {
	// Migration 52 stays as an append-only tail because local alpha DBs may have already
	// applied the task_05 loop-run schema migration before task_07 added output blobs.
	if _, err := tx.ExecContext(ctx, loopOutputBlobsTableStatement); err != nil {
		return fmt.Errorf("store: create loop output blobs table: %w", err)
	}
	return nil
}
