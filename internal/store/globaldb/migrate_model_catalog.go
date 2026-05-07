package globaldb

import (
	"context"
	"database/sql"
	"fmt"
)

func migrateModelCatalogPersistence(ctx context.Context, tx *sql.Tx) error {
	for _, statement := range modelCatalogSchemaStatements() {
		if _, err := tx.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("store: apply model catalog schema: %w", err)
		}
	}
	return nil
}
