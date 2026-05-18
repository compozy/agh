package store

import (
	"strings"
	"testing"
	"time"

	"github.com/pedronauck/agh/internal/testutil"
)

func TestRunMigrationsAppliedRegistryClawpatch(t *testing.T) {
	t.Parallel()

	t.Run("Should reject applied migrations ahead of the current registry", func(t *testing.T) {
		t.Parallel()

		ctx := testutil.Context(t)
		db := openMigrationTestDB(t, "applied-ahead.db")
		migrations := clawpatchSchemaMigrations()
		if err := RunMigrations(ctx, db, migrations); err != nil {
			t.Fatalf("RunMigrations(full registry) error = %v", err)
		}

		err := RunMigrations(ctx, db, migrations[:1])
		if err == nil || !strings.Contains(err.Error(), "applied migration 2") ||
			!strings.Contains(err.Error(), "current registry") {
			t.Fatalf("RunMigrations(truncated registry) error = %v, want applied migration registry error", err)
		}
	})

	t.Run("Should reject manually recorded unknown applied migrations", func(t *testing.T) {
		t.Parallel()

		ctx := testutil.Context(t)
		db := openMigrationTestDB(t, "applied-unknown.db")
		migrations := clawpatchSchemaMigrations()[:1]
		if err := RunMigrations(ctx, db, migrations); err != nil {
			t.Fatalf("RunMigrations(base registry) error = %v", err)
		}
		if _, err := db.ExecContext(
			ctx,
			`INSERT INTO schema_migrations (version, name, checksum, applied_at) VALUES (?, ?, ?, ?)`,
			3,
			"unknown_applied",
			"unknown-checksum",
			FormatTimestamp(time.Date(2026, 5, 17, 12, 0, 0, 0, time.UTC)),
		); err != nil {
			t.Fatalf("ExecContext(insert unknown migration) error = %v", err)
		}

		err := RunMigrations(ctx, db, migrations)
		if err == nil || !strings.Contains(err.Error(), "applied migration 3") ||
			!strings.Contains(err.Error(), "current registry") {
			t.Fatalf("RunMigrations(unknown applied) error = %v, want applied migration registry error", err)
		}
	})
}

func clawpatchSchemaMigrations() []Migration {
	return []Migration{
		{
			Version:    1,
			Name:       "create_clawpatch_schema_table",
			Statements: []string{`CREATE TABLE clawpatch_schema (id TEXT PRIMARY KEY);`},
		},
		{
			Version:    2,
			Name:       "add_clawpatch_schema_value",
			Statements: []string{`ALTER TABLE clawpatch_schema ADD COLUMN value TEXT;`},
		},
	}
}
