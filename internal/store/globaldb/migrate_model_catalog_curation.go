package globaldb

import (
	"context"
	"database/sql"
)

func migrateModelCatalogCuration(ctx context.Context, tx *sql.Tx) error {
	return addMissingMigrationColumns(ctx, tx, "model_catalog_rows", []migrationColumnSpec{
		{
			name: "deprecated",
			sql: `ALTER TABLE model_catalog_rows ADD COLUMN deprecated INTEGER NOT NULL DEFAULT 0 ` +
				`CHECK (deprecated IN (0, 1))`,
		},
		{
			name: "hidden",
			sql: `ALTER TABLE model_catalog_rows ADD COLUMN hidden INTEGER NOT NULL DEFAULT 0 ` +
				`CHECK (hidden IN (0, 1))`,
		},
		{
			name: "featured",
			sql: `ALTER TABLE model_catalog_rows ADD COLUMN featured INTEGER NOT NULL DEFAULT 0 ` +
				`CHECK (featured IN (0, 1))`,
		},
		{
			name: "release_date",
			sql:  `ALTER TABLE model_catalog_rows ADD COLUMN release_date TEXT`,
		},
	})
}
