package globaldb

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/compozy/agh/internal/store"
)

const (
	sessionCatalogRecentIndex   = "idx_sessions_catalog_recent"
	sessionCatalogActivityIndex = "idx_sessions_catalog_activity"
)

var sessionCatalogPagingMigration = store.Migration{
	Version:  68,
	Name:     "add_session_catalog_paging_indexes",
	Up:       migrateSessionCatalogPagingIndexes,
	Checksum: "2026-07-10-add-session-catalog-paging-indexes",
}

func migrateSessionCatalogPagingIndexes(ctx context.Context, tx *sql.Tx) error {
	exists, err := tableExists(ctx, tx, "sessions")
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}
	statements := []string{
		`CREATE INDEX IF NOT EXISTS idx_sessions_catalog_recent
			ON sessions(workspace_id, state, updated_at DESC, created_at DESC, id DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_sessions_catalog_activity
			ON sessions(
				workspace_id, state, COALESCE(last_update_at, updated_at) DESC,
				updated_at DESC, created_at DESC, id DESC
			)`,
	}
	for _, statement := range statements {
		if _, err := tx.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("store: add session catalog paging index: %w", err)
		}
	}
	return nil
}
