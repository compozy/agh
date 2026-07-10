package globaldb

import (
	"context"
	"database/sql"
	"fmt"
)

const taskEventsTypeSeqIndexStatement = `
CREATE INDEX IF NOT EXISTS idx_task_events_type_seq
ON task_events(event_type, event_seq);`

func migrateTaskEventsTypeSeqIndex(ctx context.Context, tx *sql.Tx) error {
	if _, err := tx.ExecContext(ctx, taskEventsTypeSeqIndexStatement); err != nil {
		return fmt.Errorf("store: create task events type sequence index: %w", err)
	}
	return nil
}
