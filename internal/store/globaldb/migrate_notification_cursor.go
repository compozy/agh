package globaldb

import (
	"context"
	"database/sql"
	"fmt"
)

func migrateNotificationCursors(ctx context.Context, tx *sql.Tx) error {
	for _, statement := range notificationCursorSchemaStatements() {
		if _, err := tx.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("store: apply notification cursor schema: %w", err)
		}
	}
	return nil
}
