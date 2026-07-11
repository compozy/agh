package globaldb

import (
	"context"
	"database/sql"
)

func migrateModelCatalogExplicitCuration(ctx context.Context, tx *sql.Tx) error {
	return addMissingMigrationColumns(ctx, tx, "model_catalog_rows", []migrationColumnSpec{
		{
			name: "explicitly_curated",
			sql: `ALTER TABLE model_catalog_rows ADD COLUMN explicitly_curated INTEGER NOT NULL DEFAULT 0 ` +
				`CHECK (explicitly_curated IN (0, 1))`,
		},
	})
}
