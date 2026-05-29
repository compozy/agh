package globaldb

import (
	"context"
	"database/sql"
	"fmt"
)

// migrateTaskRunStarvation adds the durable per-run escalation budget the convergence backstop
// advances each cycle. It is a pure additive child table (run_id -> task_runs ON DELETE CASCADE)
// with no foreign-key toggle, so it runs in the normal in-transaction Up path. Every timestamp
// column is written via store.FormatTimestamp at the service layer; none default to CURRENT_TIMESTAMP.
func migrateTaskRunStarvation(ctx context.Context, tx *sql.Tx) error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS task_run_starvation (
			run_id             TEXT NOT NULL PRIMARY KEY REFERENCES task_runs(id) ON DELETE CASCADE,
			wake_count         INTEGER NOT NULL DEFAULT 0 CHECK (wake_count >= 0),
			first_starved_at   TEXT NOT NULL,
			last_wake_at       TEXT,
			escalation_tier    INTEGER NOT NULL DEFAULT 0 CHECK (escalation_tier >= 0),
			spawn_requested_at TEXT,
			starved_event_at   TEXT,
			updated_at         TEXT NOT NULL
		);`,
		`CREATE INDEX IF NOT EXISTS idx_task_run_starvation_tier
			ON task_run_starvation(escalation_tier, run_id);`,
	}
	for _, stmt := range statements {
		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("store: create task_run_starvation: %w", err)
		}
	}
	return nil
}
