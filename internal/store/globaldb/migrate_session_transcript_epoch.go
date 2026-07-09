package globaldb

import (
	"context"
	"database/sql"
)

func migrateSessionTranscriptEpoch(ctx context.Context, tx *sql.Tx) error {
	exists, err := tableExists(ctx, tx, "sessions")
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}
	return addMissingMigrationColumns(ctx, tx, "sessions", []migrationColumnSpec{
		{
			name: sessionTranscriptEpochColumn,
			sql:  `ALTER TABLE sessions ADD COLUMN ` + sessionTranscriptEpochColumn + ` INTEGER NOT NULL DEFAULT 0`,
		},
	})
}
