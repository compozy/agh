package globaldb

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/compozy/agh/internal/store"
)

// Migration 75: persist the authenticated actor for deferred Loop control.
//
// Why: Goal Pause settles at a later prompt boundary and must retain its initiating actor.
// Affects: agh.db table loop_runs.
// Idempotent: yes; guarded column additions tolerate a registry retry.
// Reversible: no; the columns are durable audit identity.
var loopRunControlActorMigration = store.Migration{
	Version:  75,
	Name:     "add_loop_run_control_actor",
	Up:       migrateLoopRunControlActor,
	Checksum: "2026-07-10-add-loop-run-control-actor",
}

func migrateLoopRunControlActor(ctx context.Context, tx *sql.Tx) error {
	if err := addMissingMigrationColumns(ctx, tx, "loop_runs", []migrationColumnSpec{
		{name: columnControlActorKind, sql: `ALTER TABLE loop_runs ADD COLUMN control_actor_kind TEXT`},
		{name: columnControlActorID, sql: `ALTER TABLE loop_runs ADD COLUMN control_actor_id TEXT`},
		{name: columnControlRequested, sql: `ALTER TABLE loop_runs ADD COLUMN control_requested_at TIMESTAMP`},
	}); err != nil {
		return fmt.Errorf("store: add loop run control actor columns: %w", err)
	}
	return nil
}
