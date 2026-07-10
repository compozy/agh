package globaldb

import (
	"context"
	"database/sql"
	"fmt"
)

func migrateModelCatalogCurationPresence(ctx context.Context, tx *sql.Tx) error {
	if err := addMissingMigrationColumns(ctx, tx, "model_catalog_rows", []migrationColumnSpec{
		{
			name: "deprecated_set",
			sql: `ALTER TABLE model_catalog_rows ADD COLUMN deprecated_set INTEGER NOT NULL DEFAULT 0 ` +
				`CHECK (deprecated_set IN (0, 1))`,
		},
		{
			name: "hidden_set",
			sql: `ALTER TABLE model_catalog_rows ADD COLUMN hidden_set INTEGER NOT NULL DEFAULT 0 ` +
				`CHECK (hidden_set IN (0, 1))`,
		},
		{
			name: "featured_set",
			sql: `ALTER TABLE model_catalog_rows ADD COLUMN featured_set INTEGER NOT NULL DEFAULT 0 ` +
				`CHECK (featured_set IN (0, 1))`,
		},
	}); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE model_catalog_rows SET
		deprecated_set = CASE
			WHEN explicitly_curated = 1 THEN 1
			WHEN deprecated = 1 THEN 1
			ELSE deprecated_set
		END,
		hidden_set = CASE
			WHEN explicitly_curated = 1 THEN 1
			WHEN hidden = 1 THEN 1
			ELSE hidden_set
		END,
		featured_set = CASE
			WHEN explicitly_curated = 1 THEN 1
			WHEN featured = 1 THEN 1
			ELSE featured_set
		END`); err != nil {
		return fmt.Errorf("store: backfill model catalog curation presence: %w", err)
	}
	return nil
}
