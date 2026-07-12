package globaldb

import (
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/compozy/agh/internal/store"
	"github.com/compozy/agh/internal/testutil"
)

func TestLoopRunControlActorMigrationFreshDB(t *testing.T) {
	t.Run("Should install deferred control actor columns on a fresh database", func(t *testing.T) {
		t.Parallel()

		globalDB := openTestGlobalDB(t)
		for _, column := range []string{"control_actor_kind", "control_actor_id", "control_requested_at"} {
			assertTableHasColumn(t, globalDB.db, "loop_runs", column)
		}
	})
}

func TestLoopRunControlActorMigrationReopenAfterRestart(t *testing.T) {
	t.Run("Should preserve existing Loop rows while upgrading and reopening", func(t *testing.T) {
		t.Parallel()

		ctx := testutil.Context(t)
		path := filepath.Join(t.TempDir(), GlobalDatabaseName)
		db, err := store.OpenSQLiteDatabase(ctx, path, nil)
		if err != nil {
			t.Fatalf("OpenSQLiteDatabase(v64) error = %v", err)
		}
		migrationIndex := slices.IndexFunc(globalSchemaMigrations, func(migration store.Migration) bool {
			return migration.Version == loopRunControlActorMigration.Version
		})
		if migrationIndex < 0 {
			t.Fatal("loop-run control actor migration missing from registry")
		}
		if err := store.RunMigrations(ctx, db, globalSchemaMigrations[:migrationIndex]); err != nil {
			t.Fatalf("RunMigrations(v64) error = %v", err)
		}
		now := store.FormatTimestamp(time.Date(2026, 7, 10, 21, 0, 0, 0, time.UTC))
		if _, err := db.ExecContext(
			ctx,
			`INSERT INTO workspaces (id, root_dir, name, created_at, updated_at)
			 VALUES ('ws-control-actor', '/tmp/ws-control-actor', 'Control actor', ?, ?)`,
			now,
			now,
		); err != nil {
			t.Fatalf("insert workspace error = %v", err)
		}
		if _, err := db.ExecContext(
			ctx,
			`INSERT INTO loop_runs (
				id, workspace_id, loop_name, status, last_progress_at, inputs_json
			) VALUES ('run-control-actor', 'ws-control-actor', 'delivery', 'running', ?, '{}')`,
			now,
		); err != nil {
			t.Fatalf("insert v64 Loop row error = %v", err)
		}
		if err := db.Close(); err != nil {
			t.Fatalf("Close(v64) error = %v", err)
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
		for _, column := range []string{"control_actor_kind", "control_actor_id", "control_requested_at"} {
			assertTableHasColumn(t, reopened.db, "loop_runs", column)
		}
		var status string
		if err := reopened.db.QueryRowContext(
			ctx,
			`SELECT status FROM loop_runs WHERE id = 'run-control-actor'`,
		).Scan(&status); err != nil {
			t.Fatalf("query preserved Loop row error = %v", err)
		}
		if status != "running" {
			t.Fatalf("preserved Loop status = %q, want running", status)
		}
	})
}
