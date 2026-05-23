package workspacedb

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"

	"github.com/compozy/agh/internal/store"
	"github.com/compozy/agh/internal/testutil"
	aghworkspace "github.com/compozy/agh/internal/workspace"
)

func TestOpen(t *testing.T) {
	t.Run("Should create workspace identity database and run migrations once", func(t *testing.T) {
		t.Parallel()

		ctx := testutil.Context(t)
		workspaceRoot := t.TempDir()
		db := openWorkspaceTestDB(ctx, t, workspaceRoot, workspaceTestMigrations())

		realWorkspaceRoot, err := filepath.EvalSymlinks(workspaceRoot)
		if err != nil {
			t.Fatalf("EvalSymlinks(workspaceRoot) error = %v", err)
		}
		if got, want := db.Path(), filepath.Join(realWorkspaceRoot, ".agh", store.GlobalDatabaseName); got != want {
			t.Fatalf("Path() = %q, want %q", got, want)
		}
		if !aghworkspace.IsWorkspaceID(db.WorkspaceID()) {
			t.Fatalf("WorkspaceID() = %q, want canonical ULID", db.WorkspaceID())
		}

		if _, err := db.DB().ExecContext(ctx, `INSERT INTO records (id) VALUES ('first')`); err != nil {
			t.Fatalf("Insert first record error = %v", err)
		}
		if err := db.Close(ctx); err != nil {
			t.Fatalf("Close(first) error = %v", err)
		}

		reopened := openWorkspaceTestDB(ctx, t, workspaceRoot, workspaceTestMigrations())
		var migrationRuns int
		if err := reopened.DB().
			QueryRowContext(ctx, `SELECT COUNT(*) FROM migration_runs`).
			Scan(&migrationRuns); err != nil {
			t.Fatalf("Query migration_runs error = %v", err)
		}
		if migrationRuns != 1 {
			t.Fatalf("migration_runs count = %d, want 1", migrationRuns)
		}
		var records int
		if err := reopened.DB().QueryRowContext(ctx, `SELECT COUNT(*) FROM records`).Scan(&records); err != nil {
			t.Fatalf("Query records error = %v", err)
		}
		if records != 1 {
			t.Fatalf("records count = %d, want persisted row", records)
		}
	})

	t.Run("Should reject workspace databases ahead of the binary migration head", func(t *testing.T) {
		t.Parallel()

		ctx := testutil.Context(t)
		workspaceRoot := t.TempDir()
		db := openWorkspaceTestDB(ctx, t, workspaceRoot, workspaceTestMigrations())
		if _, err := db.DB().ExecContext(
			ctx,
			`INSERT INTO schema_migrations (version, name, checksum, applied_at)
			 VALUES (99, 'future_schema', 'future', '2026-05-05T00:00:00.000000000Z')`,
		); err != nil {
			t.Fatalf("Insert future migration error = %v", err)
		}
		if err := db.Close(ctx); err != nil {
			t.Fatalf("Close() error = %v", err)
		}

		_, err := Open(ctx, Options{WorkspaceRoot: workspaceRoot, Migrations: workspaceTestMigrations()})
		if !errors.Is(err, errAheadSchema) {
			t.Fatalf("Open(ahead schema) error = %v, want errAheadSchema", err)
		}
	})

	t.Run("Should isolate rows across workspace database files", func(t *testing.T) {
		t.Parallel()

		ctx := testutil.Context(t)
		first := openWorkspaceTestDB(ctx, t, t.TempDir(), workspaceTestMigrations())
		second := openWorkspaceTestDB(ctx, t, t.TempDir(), workspaceTestMigrations())

		if _, err := first.DB().ExecContext(ctx, `INSERT INTO records (id) VALUES ('first')`); err != nil {
			t.Fatalf("Insert first workspace record error = %v", err)
		}
		if _, err := second.DB().ExecContext(ctx, `INSERT INTO records (id) VALUES ('second')`); err != nil {
			t.Fatalf("Insert second workspace record error = %v", err)
		}

		assertWorkspaceRecordCount(ctx, t, first.DB(), "first", 1)
		assertWorkspaceRecordCount(ctx, t, first.DB(), "second", 0)
		assertWorkspaceRecordCount(ctx, t, second.DB(), "second", 1)
		assertWorkspaceRecordCount(ctx, t, second.DB(), "first", 0)
	})

	t.Run("Should support OpenWorkspace helper and idempotent close", func(t *testing.T) {
		t.Parallel()

		ctx := testutil.Context(t)
		workspaceRoot := t.TempDir()
		db, err := OpenWorkspace(ctx, workspaceRoot, workspaceTestMigrations())
		if err != nil {
			t.Fatalf("OpenWorkspace() error = %v", err)
		}

		if db.DB() == nil {
			t.Fatal("DB() = nil, want SQL handle")
		}
		if db.WorkspaceRoot() == "" {
			t.Fatal("WorkspaceRoot() = empty, want configured root")
		}
		if db.Path() == "" {
			t.Fatal("Path() = empty, want database path")
		}
		if db.WorkspaceID() == "" {
			t.Fatal("WorkspaceID() = empty, want resolved identity")
		}
		if err := db.Close(ctx); err != nil {
			t.Fatalf("Close(first) error = %v", err)
		}
		if err := db.Close(ctx); err != nil {
			t.Fatalf("Close(second) error = %v", err)
		}
	})

	t.Run("Should reject invalid open and close inputs", func(t *testing.T) {
		t.Parallel()

		ctx := testutil.Context(t)
		if _, err := Open(ctx, Options{WorkspaceRoot: "   ", Migrations: workspaceTestMigrations()}); err == nil {
			t.Fatal("Open(blank root) error = nil, want validation error")
		}

		var nilDB *DB
		if nilDB.Path() != "" {
			t.Fatal("nil DB Path() returned non-empty path")
		}
		if nilDB.WorkspaceID() != "" {
			t.Fatal("nil DB WorkspaceID() returned non-empty ID")
		}
		if nilDB.WorkspaceRoot() != "" {
			t.Fatal("nil DB WorkspaceRoot() returned non-empty root")
		}
		if nilDB.DB() != nil {
			t.Fatal("nil DB DB() returned non-nil SQL handle")
		}
		if err := nilDB.Close(ctx); err != nil {
			t.Fatalf("nil DB Close() error = %v", err)
		}
	})

	t.Run("Should use an isolated custom migration table", func(t *testing.T) {
		t.Parallel()

		ctx := testutil.Context(t)
		db := openWorkspaceTestDBWithOptions(ctx, t, Options{
			WorkspaceRoot:   t.TempDir(),
			Migrations:      workspaceTestMigrations(),
			MigrationsTable: "memv2_schema_migrations",
		})

		records, err := store.AppliedMigrationsWithTable(ctx, db.DB(), "memv2_schema_migrations")
		if err != nil {
			t.Fatalf("AppliedMigrationsWithTable(custom) error = %v", err)
		}
		if got, want := len(records), len(workspaceTestMigrations()); got != want {
			t.Fatalf("custom migration records = %d, want %d", got, want)
		}
		defaultRecords, err := store.AppliedMigrations(ctx, db.DB())
		if err != nil {
			t.Fatalf("AppliedMigrations(default) error = %v", err)
		}
		if len(defaultRecords) != 0 {
			t.Fatalf("default migration records = %d, want 0", len(defaultRecords))
		}
	})
}

func workspaceTestMigrations() []store.Migration {
	return []store.Migration{
		{
			Version:    1,
			Name:       "create_records",
			Statements: []string{`CREATE TABLE records (id TEXT PRIMARY KEY);`},
		},
		{
			Version:  2,
			Name:     "record_migration_run",
			Checksum: "workspace-test-record-run-v1",
			Up: func(ctx context.Context, tx *sql.Tx) error {
				if _, err := tx.ExecContext(ctx, `CREATE TABLE migration_runs (id INTEGER PRIMARY KEY)`); err != nil {
					return err
				}
				_, err := tx.ExecContext(ctx, `INSERT INTO migration_runs DEFAULT VALUES`)
				return err
			},
		},
	}
}

func openWorkspaceTestDB(
	ctx context.Context,
	t *testing.T,
	workspaceRoot string,
	migrations []store.Migration,
) *DB {
	t.Helper()

	return openWorkspaceTestDBWithOptions(ctx, t, Options{WorkspaceRoot: workspaceRoot, Migrations: migrations})
}

func openWorkspaceTestDBWithOptions(ctx context.Context, t *testing.T, opts Options) *DB {
	t.Helper()

	db, err := Open(ctx, opts)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() {
		if err := db.Close(ctx); err != nil {
			t.Fatalf("DB.Close() error = %v", err)
		}
	})
	return db
}

func assertWorkspaceRecordCount(ctx context.Context, t *testing.T, db *sql.DB, id string, want int) {
	t.Helper()

	var count int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM records WHERE id = ?`, id).Scan(&count); err != nil {
		t.Fatalf("Query record count for %q error = %v", id, err)
	}
	if count != want {
		t.Fatalf("record count for %q = %d, want %d", id, count, want)
	}
}
