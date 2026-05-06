package globaldb

import (
	"context"
	"database/sql"
	"fmt"
)

func migrateBridgeTaskSubscriptions(ctx context.Context, tx *sql.Tx) error {
	for _, statement := range bridgeTaskSubscriptionSchemaStatements() {
		if _, err := tx.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("store: create bridge task subscriptions schema: %w", err)
		}
	}
	return nil
}
