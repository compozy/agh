package globaldb

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/compozy/agh/internal/store"
)

// Migration 72: remove the workspace/state index subsumed by the recent-catalog index.
// Why: the covering catalog index owns the same prefix plus the public recent ordering.
// Affects: session catalog index topology only; no rows or public contracts change.
// Idempotent: yes; DROP INDEX IF EXISTS is safe on fresh and upgraded databases.
// Reversible: yes; v61 contains the superseded index definition if restoration is required.
var sessionsCatalogIndexCleanupMigration = store.Migration{
	Version:  72,
	Name:     "drop_redundant_sessions_workspace_state_index",
	Up:       migrateSessionsCatalogIndexCleanup,
	Checksum: "2026-07-11-drop-redundant-sessions-workspace-state-index",
}

func migrateSessionsCatalogIndexCleanup(ctx context.Context, tx *sql.Tx) error {
	if _, err := tx.ExecContext(ctx, `DROP INDEX IF EXISTS idx_sessions_workspace_state`); err != nil {
		return fmt.Errorf("store: drop redundant sessions workspace/state index: %w", err)
	}
	return nil
}
