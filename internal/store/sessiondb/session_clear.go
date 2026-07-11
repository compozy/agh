package sessiondb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/compozy/agh/internal/store"
)

func clearSessionSQLite(ctx context.Context, db *sql.DB) error {
	if ctx == nil {
		return errors.New("store: clear session sqlite context is required")
	}
	if db == nil {
		return errors.New("store: clear session sqlite database is required")
	}
	return store.ExecuteWriteNoCheckpoint(ctx, db, func(ctx context.Context, tx *store.WriteTx) error {
		for _, statement := range []string{
			"DELETE FROM hook_runs",
			"DELETE FROM token_usage",
			"DELETE FROM transcript_tool_routes",
			"DELETE FROM transcript_entries",
			"DELETE FROM events",
		} {
			if _, err := tx.ExecContext(ctx, statement); err != nil {
				return fmt.Errorf("store: clear session table: %w", err)
			}
		}
		if _, err := tx.ExecContext(ctx, `
			UPDATE transcript_projection_state
			SET generation = generation + 1, active_entry_key = NULL
			WHERE singleton = 1`); err != nil {
			return fmt.Errorf("store: advance transcript projection generation: %w", err)
		}
		return nil
	})
}
