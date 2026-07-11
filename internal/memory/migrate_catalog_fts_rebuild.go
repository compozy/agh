package memory

import (
	"context"
	"database/sql"
	"fmt"
)

func migrateCatalogFTSRebuild(ctx context.Context, tx *sql.Tx) error {
	if _, err := tx.ExecContext(
		ctx,
		`INSERT INTO memory_catalog_fts(memory_catalog_fts) VALUES ('rebuild')`,
	); err != nil {
		return fmt.Errorf("memory: rebuild catalog FTS index: %w", err)
	}
	return nil
}
