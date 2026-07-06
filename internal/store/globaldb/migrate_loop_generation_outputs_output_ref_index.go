package globaldb

import (
	"context"
	"database/sql"
	"fmt"
)

func migrateLoopGenerationOutputsOutputRefIndex(ctx context.Context, tx *sql.Tx) error {
	if _, err := tx.ExecContext(ctx, loopGenerationOutputsOutputRefIndexStatement); err != nil {
		return fmt.Errorf("store: create loop generation outputs output_ref index: %w", err)
	}
	return nil
}
