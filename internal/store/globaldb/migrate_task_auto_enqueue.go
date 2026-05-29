package globaldb

import (
	"context"
	"database/sql"
)

const migrateTaskAutoEnqueueOnReadyKey = "auto_enqueue_on_ready"

func migrateTaskAutoEnqueueSchema(ctx context.Context, tx *sql.Tx) error {
	exists, err := tableExists(ctx, tx, "tasks")
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}
	return addMissingMigrationColumns(ctx, tx, "tasks", taskAutoEnqueueColumnSpecs())
}

func taskAutoEnqueueColumnSpecs() []migrationColumnSpec {
	return []migrationColumnSpec{
		{
			name: migrateTaskAutoEnqueueOnReadyKey,
			sql: `ALTER TABLE tasks ADD COLUMN auto_enqueue_on_ready INTEGER NOT NULL DEFAULT 0 ` +
				`CHECK (auto_enqueue_on_ready IN (0, 1))`,
		},
	}
}
