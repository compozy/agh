package globaldb

import (
	"context"
	"database/sql"
)

func migrateLoopConfigModelDefaults(ctx context.Context, tx *sql.Tx) error {
	return addMissingMigrationColumns(ctx, tx, "loop_config", []migrationColumnSpec{
		{
			name: "model_default_worker",
			sql:  "ALTER TABLE loop_config ADD COLUMN model_default_worker TEXT",
		},
		{
			name: "model_default_judge",
			sql:  "ALTER TABLE loop_config ADD COLUMN model_default_judge TEXT",
		},
	})
}
