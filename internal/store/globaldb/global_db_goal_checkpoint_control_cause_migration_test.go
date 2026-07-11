package globaldb

import (
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/compozy/agh/internal/store"
	"github.com/compozy/agh/internal/testutil"
)

func TestGoalCheckpointControlCauseMigrationFreshDB(t *testing.T) {
	t.Run("Should install the exact checkpoint control cause column on a fresh database", func(t *testing.T) {
		t.Parallel()

		globalDB := openTestGlobalDB(t)
		assertTableHasColumn(t, globalDB.db, "loop_goal_checkpoints", "control_cause")
	})
}

func TestGoalCheckpointControlCauseMigrationReopenAfterRestart(t *testing.T) {
	t.Run("Should preserve an awaiting checkpoint while upgrading and reopening", func(t *testing.T) {
		t.Parallel()

		ctx := testutil.Context(t)
		path := filepath.Join(t.TempDir(), GlobalDatabaseName)
		db, err := store.OpenSQLiteDatabase(ctx, path, nil)
		if err != nil {
			t.Fatalf("OpenSQLiteDatabase(v65) error = %v", err)
		}
		migrationIndex := slices.IndexFunc(globalSchemaMigrations, func(migration store.Migration) bool {
			return migration.Version == goalCheckpointControlCauseMigration.Version
		})
		if migrationIndex < 0 {
			t.Fatal("Goal checkpoint control-cause migration missing from registry")
		}
		if err := store.RunMigrations(ctx, db, globalSchemaMigrations[:migrationIndex]); err != nil {
			t.Fatalf("RunMigrations(v65) error = %v", err)
		}
		now := store.FormatTimestamp(time.Date(2026, 7, 10, 22, 0, 0, 0, time.UTC))
		if _, err := db.ExecContext(
			ctx,
			`INSERT INTO workspaces (id, root_dir, name, created_at, updated_at)
			 VALUES ('ws-control-cause', '/tmp/ws-control-cause', 'Control cause', ?, ?)`,
			now,
			now,
		); err != nil {
			t.Fatalf("insert workspace error = %v", err)
		}
		if _, err := db.ExecContext(
			ctx,
			`INSERT INTO loop_runs (
				id, workspace_id, loop_name, status, last_progress_at, inputs_json
			) VALUES ('run-control-cause', 'ws-control-cause', 'delivery', 'paused', ?, '{}')`,
			now,
		); err != nil {
			t.Fatalf("insert v65 Loop row error = %v", err)
		}
		if _, err := db.ExecContext(
			ctx,
			`INSERT INTO loop_goal_checkpoints (
				loop_run_id, generation, node_id, phase, goal_status, turn_limit,
				context_nudge_ratio, updated_at
			) VALUES ('run-control-cause', 1, 'converge', 'awaiting_control', 'paused', 3, 0.8, ?)`,
			now,
		); err != nil {
			t.Fatalf("insert v65 checkpoint error = %v", err)
		}
		if err := db.Close(); err != nil {
			t.Fatalf("Close(v65) error = %v", err)
		}

		globalDB, err := OpenGlobalDB(ctx, path)
		if err != nil {
			t.Fatalf("OpenGlobalDB(upgrade) error = %v", err)
		}
		if err := globalDB.Close(ctx); err != nil {
			t.Fatalf("Close(upgraded) error = %v", err)
		}
		reopened, err := OpenGlobalDB(ctx, path)
		if err != nil {
			t.Fatalf("OpenGlobalDB(reopen) error = %v", err)
		}
		t.Cleanup(func() {
			if err := reopened.Close(testutil.Context(t)); err != nil {
				t.Errorf("Close(reopened) error = %v", err)
			}
		})
		assertTableHasColumn(t, reopened.db, "loop_goal_checkpoints", "control_cause")

		var phase, status string
		var cause *string
		if err := reopened.db.QueryRowContext(
			ctx,
			`SELECT phase, goal_status, control_cause
			 FROM loop_goal_checkpoints
			 WHERE loop_run_id = 'run-control-cause' AND generation = 1 AND node_id = 'converge'`,
		).Scan(&phase, &status, &cause); err != nil {
			t.Fatalf("query preserved checkpoint error = %v", err)
		}
		if phase != "awaiting_control" || status != "paused" || cause != nil {
			t.Fatalf("preserved checkpoint = phase:%q status:%q cause:%v", phase, status, cause)
		}
	})
}
