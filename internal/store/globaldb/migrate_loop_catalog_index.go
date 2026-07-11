package globaldb

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/compozy/agh/internal/store"
)

const loopRunCatalogIndex = "idx_loop_runs_catalog"

var loopCatalogPagingMigration = store.Migration{
	Version:  69,
	Name:     "add_loop_catalog_run_index",
	Up:       migrateLoopCatalogRunIndex,
	Checksum: "2026-07-10-add-loop-catalog-run-index",
}

func migrateLoopCatalogRunIndex(ctx context.Context, tx *sql.Tx) error {
	exists, err := tableExists(ctx, tx, "loop_runs")
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}
	if _, err := tx.ExecContext(
		ctx,
		`CREATE INDEX IF NOT EXISTS idx_loop_runs_catalog
			ON loop_runs(workspace_id, loop_name, created_at DESC, id DESC, status)`,
	); err != nil {
		return fmt.Errorf("store: add loop catalog run index: %w", err)
	}
	return nil
}
