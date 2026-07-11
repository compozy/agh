package globaldb

import (
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/compozy/agh/internal/store"
	"github.com/compozy/agh/internal/testutil"
)

func TestLoopRunOriginIdentityMigrationFreshDB(t *testing.T) {
	t.Run("Should install every immutable inline origin field on a fresh database", func(t *testing.T) {
		t.Parallel()

		globalDB := openTestGlobalDB(t)
		for _, column := range []string{
			"origin_creation_profile_ref",
			"origin_policy_spec_digest",
			"origin_creation_digest",
		} {
			assertTableHasColumn(t, globalDB.db, "loop_runs", column)
		}
	})
}

func TestLoopRunOriginIdentityMigrationReopenAfterRestart(t *testing.T) {
	t.Run("Should preserve pre-migration catalog Runs through upgrade and reopen", func(t *testing.T) {
		t.Parallel()

		ctx := testutil.Context(t)
		path := filepath.Join(t.TempDir(), GlobalDatabaseName)
		db, err := store.OpenSQLiteDatabase(ctx, path, nil)
		if err != nil {
			t.Fatalf("OpenSQLiteDatabase(v67) error = %v", err)
		}
		migrationIndex := slices.IndexFunc(globalSchemaMigrations, func(migration store.Migration) bool {
			return migration.Version == loopRunOriginIdentityMigration.Version
		})
		if migrationIndex < 0 {
			t.Fatal("Loop Run origin identity migration missing from registry")
		}
		if err := store.RunMigrations(ctx, db, globalSchemaMigrations[:migrationIndex]); err != nil {
			t.Fatalf("RunMigrations(v67) error = %v", err)
		}
		now := store.FormatTimestamp(time.Date(2026, 7, 10, 22, 30, 0, 0, time.UTC))
		if _, err := db.ExecContext(
			ctx,
			`INSERT INTO workspaces (id, root_dir, name, created_at, updated_at)
			 VALUES ('ws-origin-migration', '/tmp/ws-origin-migration', 'Origin migration', ?, ?)`,
			now,
			now,
		); err != nil {
			t.Fatalf("insert workspace error = %v", err)
		}
		if _, err := db.ExecContext(
			ctx,
			`INSERT INTO loop_runs (
				id, workspace_id, loop_name, status, last_progress_at, inputs_json, origin_kind
			) VALUES ('run-origin-migration', 'ws-origin-migration', 'delivery', 'done', ?, '{}', 'catalog')`,
			now,
		); err != nil {
			t.Fatalf("insert v67 Loop Run error = %v", err)
		}
		if err := db.Close(); err != nil {
			t.Fatalf("Close(v67) error = %v", err)
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

		var originKind string
		var profileRef, policyDigest, creationDigest *string
		if err := reopened.db.QueryRowContext(
			ctx,
			`SELECT origin_kind, origin_creation_profile_ref, origin_policy_spec_digest, origin_creation_digest
			 FROM loop_runs WHERE id = 'run-origin-migration'`,
		).Scan(&originKind, &profileRef, &policyDigest, &creationDigest); err != nil {
			t.Fatalf("query preserved Loop Run error = %v", err)
		}
		if originKind != "catalog" || profileRef != nil || policyDigest != nil || creationDigest != nil {
			t.Fatalf(
				"preserved origin = kind:%q profile:%v policy:%v creation:%v",
				originKind,
				profileRef,
				policyDigest,
				creationDigest,
			)
		}
	})
}
