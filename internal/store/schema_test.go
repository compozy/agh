package store

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/compozy/agh/internal/testutil"
)

func TestRunMigrationsAppliesOrderedMigrationsOnce(t *testing.T) {
	t.Parallel()

	ctx := testutil.Context(t)
	db := openMigrationTestDB(t, "ordered.db")
	migrations := []Migration{
		{
			Version:  2,
			Name:     "insert_second",
			Checksum: "custom-v2",
			Up: func(ctx context.Context, tx *sql.Tx) error {
				_, err := tx.ExecContext(ctx, `INSERT INTO migration_order (step) VALUES ('second')`)
				return err
			},
		},
		{
			Version: 1,
			Name:    "create_order_table",
			Statements: []string{
				`CREATE TABLE migration_order (step TEXT PRIMARY KEY);`,
				`INSERT INTO migration_order (step) VALUES ('first');`,
			},
		},
	}

	if err := RunMigrations(ctx, db, migrations); err != nil {
		t.Fatalf("RunMigrations(first) error = %v", err)
	}
	if err := RunMigrations(ctx, db, migrations); err != nil {
		t.Fatalf("RunMigrations(second) error = %v", err)
	}

	if got, want := migrationOrderSteps(t, db), []string{"first", "second"}; !testutil.EqualStringSlices(got, want) {
		t.Fatalf("migration_order steps = %#v, want %#v", got, want)
	}
	records, err := AppliedMigrations(ctx, db)
	if err != nil {
		t.Fatalf("AppliedMigrations() error = %v", err)
	}
	if got, want := len(records), 2; got != want {
		t.Fatalf("len(records) = %d, want %d", got, want)
	}
	if records[0].Version != 1 || records[0].Name != "create_order_table" {
		t.Fatalf("records[0] = %#v, want version 1 create_order_table", records[0])
	}
	if records[1].Version != 2 || records[1].Name != "insert_second" {
		t.Fatalf("records[1] = %#v, want version 2 insert_second", records[1])
	}
}

func TestRunMigrationsUsesIndependentMigrationTables(t *testing.T) {
	t.Run("Should use independent migration tables for namespaced schemas", func(t *testing.T) {
		t.Parallel()

		ctx := testutil.Context(t)
		db := openMigrationTestDB(t, "namespaced.db")
		defaultMigrations := []Migration{{
			Version:    1,
			Name:       "create_default_table",
			Statements: []string{`CREATE TABLE default_owned (id TEXT PRIMARY KEY);`},
		}}
		memoryMigrations := []Migration{{
			Version:    1,
			Name:       "create_memory_table",
			Statements: []string{`CREATE TABLE memory_owned (id TEXT PRIMARY KEY);`},
		}}

		if err := RunMigrations(ctx, db, defaultMigrations); err != nil {
			t.Fatalf("RunMigrations(default) error = %v", err)
		}
		if err := RunMigrations(
			ctx,
			db,
			memoryMigrations,
			WithMigrationsTable("memory_schema_migrations"),
		); err != nil {
			t.Fatalf("RunMigrations(memory) error = %v", err)
		}
		if err := RunMigrations(ctx, db, defaultMigrations); err != nil {
			t.Fatalf("RunMigrations(default repeat) error = %v", err)
		}
		if err := RunMigrations(
			ctx,
			db,
			memoryMigrations,
			WithMigrationsTable("memory_schema_migrations"),
		); err != nil {
			t.Fatalf("RunMigrations(memory repeat) error = %v", err)
		}

		defaultRecords, err := AppliedMigrations(ctx, db)
		if err != nil {
			t.Fatalf("AppliedMigrations(default) error = %v", err)
		}
		if got, want := len(defaultRecords), 1; got != want {
			t.Fatalf("len(defaultRecords) = %d, want %d", got, want)
		}
		if defaultRecords[0].Name != "create_default_table" {
			t.Fatalf("defaultRecords[0].Name = %q, want create_default_table", defaultRecords[0].Name)
		}

		memoryRecords, err := appliedMigrations(ctx, db, "memory_schema_migrations")
		if err != nil {
			t.Fatalf("appliedMigrations(memory) error = %v", err)
		}
		if got, want := len(memoryRecords), 1; got != want {
			t.Fatalf("len(memoryRecords) = %d, want %d", got, want)
		}
		if memoryRecords[0].Name != "create_memory_table" {
			t.Fatalf("memoryRecords[0].Name = %q, want create_memory_table", memoryRecords[0].Name)
		}
	})
}

func TestRunMigrationsRejectsInvalidMigrationTableNames(t *testing.T) {
	t.Run("Should reject invalid migration table names", func(t *testing.T) {
		t.Parallel()

		ctx := testutil.Context(t)
		db := openMigrationTestDB(t, "invalid-table.db")
		migrations := []Migration{{
			Version:    1,
			Name:       "create_valid",
			Statements: []string{`CREATE TABLE valid_table_name (id TEXT PRIMARY KEY);`},
		}}

		tests := []struct {
			name  string
			table string
		}{
			{name: "empty", table: " "},
			{name: "starts with digit", table: "1_schema_migrations"},
			{name: "contains dash", table: "memory-schema-migrations"},
			{name: "contains quote", table: `memory"schema`},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				err := RunMigrations(ctx, db, migrations, WithMigrationsTable(tt.table))
				if err == nil || !strings.Contains(err.Error(), "migration table name") {
					t.Fatalf("RunMigrations() error = %v, want migration table name validation", err)
				}
			})
		}
	})
}

func TestRunMigrationsFailedMigrationRollsBackAndDoesNotRecord(t *testing.T) {
	t.Parallel()

	ctx := testutil.Context(t)
	db := openMigrationTestDB(t, "failure.db")
	sentinel := errors.New("boom")

	err := RunMigrations(ctx, db, []Migration{
		{
			Version:    1,
			Name:       "create_failure_table",
			Statements: []string{`CREATE TABLE migration_failures (step TEXT PRIMARY KEY);`},
		},
		{
			Version:  2,
			Name:     "failing_insert",
			Checksum: "failing-v2",
			Up: func(ctx context.Context, tx *sql.Tx) error {
				if _, err := tx.ExecContext(
					ctx,
					`INSERT INTO migration_failures (step) VALUES ('rolled-back')`,
				); err != nil {
					return err
				}
				return sentinel
			},
		},
	})
	if !errors.Is(err, sentinel) {
		t.Fatalf("RunMigrations() error = %v, want sentinel", err)
	}
	if !strings.Contains(err.Error(), `apply migration 2 "failing_insert"`) {
		t.Fatalf("RunMigrations() error = %v, want wrapped migration context", err)
	}

	var rows int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM migration_failures`).Scan(&rows); err != nil {
		t.Fatalf("QueryRowContext(count) error = %v", err)
	}
	if rows != 0 {
		t.Fatalf("migration_failures row count = %d, want rollback to 0", rows)
	}
	records, err := AppliedMigrations(ctx, db)
	if err != nil {
		t.Fatalf("AppliedMigrations() error = %v", err)
	}
	if got, want := len(records), 1; got != want {
		t.Fatalf("len(records) = %d, want %d", got, want)
	}
	if records[0].Version != 1 {
		t.Fatalf("records[0].Version = %d, want 1", records[0].Version)
	}
}

func TestRunMigrationsDetectsAppliedMigrationIntegrityMismatch(t *testing.T) {
	t.Parallel()

	ctx := testutil.Context(t)
	db := openMigrationTestDB(t, "integrity.db")
	original := []Migration{{
		Version:    1,
		Name:       "create_integrity_table",
		Statements: []string{`CREATE TABLE migration_integrity (id TEXT PRIMARY KEY);`},
	}}
	if err := RunMigrations(ctx, db, original); err != nil {
		t.Fatalf("RunMigrations(original) error = %v", err)
	}

	err := RunMigrations(ctx, db, []Migration{{
		Version:    1,
		Name:       "create_integrity_table",
		Statements: []string{`CREATE TABLE migration_integrity (id TEXT PRIMARY KEY, value TEXT);`},
	}})
	if err == nil || !strings.Contains(err.Error(), "integrity mismatch") {
		t.Fatalf("RunMigrations(modified) error = %v, want integrity mismatch", err)
	}
}

func TestRunMigrationsValidatesDefinitions(t *testing.T) {
	t.Parallel()

	ctx := testutil.Context(t)
	tests := []struct {
		name       string
		migrations []Migration
		wantErr    string
	}{
		{
			name:       "nil context",
			migrations: []Migration{{Version: 1, Name: "valid", Statements: []string{`CREATE TABLE valid (id TEXT);`}}},
			wantErr:    "context is required",
		},
		{
			name: "invalid version",
			migrations: []Migration{
				{Version: 0, Name: "invalid", Statements: []string{`CREATE TABLE invalid (id TEXT);`}},
			},
			wantErr: "invalid version",
		},
		{
			name:       "empty name",
			migrations: []Migration{{Version: 1, Name: " ", Statements: []string{`CREATE TABLE unnamed (id TEXT);`}}},
			wantErr:    "name is required",
		},
		{
			name: "duplicate version",
			migrations: []Migration{
				{Version: 1, Name: "first", Statements: []string{`CREATE TABLE first (id TEXT);`}},
				{Version: 1, Name: "second", Statements: []string{`CREATE TABLE second (id TEXT);`}},
			},
			wantErr: "duplicate migration version",
		},
		{
			name: "duplicate name",
			migrations: []Migration{
				{Version: 1, Name: "same", Statements: []string{`CREATE TABLE first_same (id TEXT);`}},
				{Version: 2, Name: "same", Statements: []string{`CREATE TABLE second_same (id TEXT);`}},
			},
			wantErr: "duplicate migration name",
		},
		{
			name:       "missing operation",
			migrations: []Migration{{Version: 1, Name: "noop"}},
			wantErr:    "has no operation",
		},
		{
			name: "custom operation without checksum",
			migrations: []Migration{{
				Version: 1,
				Name:    "custom",
				Up: func(context.Context, *sql.Tx) error {
					return nil
				},
			}},
			wantErr: "checksum is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			db := openMigrationTestDB(t, tt.name+".db")
			runCtx := ctx
			if tt.name == "nil context" {
				runCtx = nil
			}

			err := RunMigrations(runCtx, db, tt.migrations)
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("RunMigrations() error = %v, want substring %q", err, tt.wantErr)
			}
		})
	}
}

func TestRunMigrationsStatementFailureRollsBackAndDoesNotRecord(t *testing.T) {
	t.Parallel()

	ctx := testutil.Context(t)
	db := openMigrationTestDB(t, "statement-failure.db")
	err := RunMigrations(ctx, db, []Migration{{
		Version: 1,
		Name:    "failing_statement",
		Statements: []string{
			`CREATE TABLE statement_failures (step TEXT PRIMARY KEY);`,
			`INSERT INTO missing_statement_failures (step) VALUES ('boom');`,
		},
	}})
	if err == nil || !strings.Contains(err.Error(), `apply migration 1 "failing_statement"`) {
		t.Fatalf("RunMigrations() error = %v, want wrapped statement failure", err)
	}

	var tableCount int
	if err := db.QueryRowContext(
		ctx,
		`SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = 'statement_failures'`,
	).Scan(&tableCount); err != nil {
		t.Fatalf("QueryRowContext(table count) error = %v", err)
	}
	if tableCount != 0 {
		t.Fatalf("statement_failures table count = %d, want rollback to 0", tableCount)
	}

	records, err := AppliedMigrations(ctx, db)
	if err != nil {
		t.Fatalf("AppliedMigrations() error = %v", err)
	}
	if len(records) != 0 {
		t.Fatalf("len(records) = %d, want 0", len(records))
	}
}

func TestAppliedMigrationsHandlesMissingTableAndInvalidInputs(t *testing.T) {
	t.Parallel()

	ctx := testutil.Context(t)
	db := openMigrationTestDB(t, "applied-missing.db")
	records, err := AppliedMigrations(ctx, db)
	if err != nil {
		t.Fatalf("AppliedMigrations(missing table) error = %v", err)
	}
	if len(records) != 0 {
		t.Fatalf("len(records) = %d, want 0", len(records))
	}
	if _, err := AppliedMigrations(
		nilMigrationContext(),
		db,
	); err == nil ||
		!strings.Contains(err.Error(), "context is required") {
		t.Fatalf("AppliedMigrations(nil context) error = %v, want context error", err)
	}
	if _, err := AppliedMigrations(ctx, nil); err == nil || !strings.Contains(err.Error(), "database is required") {
		t.Fatalf("AppliedMigrations(nil db) error = %v, want database error", err)
	}
	if err := RunMigrations(ctx, nil, nil); err == nil || !strings.Contains(err.Error(), "database is required") {
		t.Fatalf("RunMigrations(nil db) error = %v, want database error", err)
	}
}

func TestAppliedMigrationsReturnsTimestampParseError(t *testing.T) {
	t.Parallel()

	ctx := testutil.Context(t)
	db := openMigrationTestDB(t, "bad-timestamp.db")
	if _, err := db.ExecContext(ctx, `CREATE TABLE schema_migrations (
		version    INTEGER PRIMARY KEY,
		name       TEXT NOT NULL,
		checksum   TEXT NOT NULL,
		applied_at TEXT NOT NULL
	);`); err != nil {
		t.Fatalf("create schema_migrations error = %v", err)
	}
	if _, err := db.ExecContext(
		ctx,
		`INSERT INTO schema_migrations (version, name, checksum, applied_at) VALUES (1, 'bad', 'sum', 'not-a-time')`,
	); err != nil {
		t.Fatalf("insert bad migration row error = %v", err)
	}

	_, err := AppliedMigrations(ctx, db)
	if err == nil || !strings.Contains(err.Error(), "parse schema migration timestamp") {
		t.Fatalf("AppliedMigrations() error = %v, want timestamp parse error", err)
	}
}

func openMigrationTestDB(t *testing.T, name string) *sql.DB {
	t.Helper()

	db, err := OpenSQLiteDatabase(testutil.Context(t), filepath.Join(t.TempDir(), name), nil)
	if err != nil {
		t.Fatalf("OpenSQLiteDatabase() error = %v", err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf("db.Close() error = %v", err)
		}
	})
	return db
}

func migrationOrderSteps(t *testing.T, db *sql.DB) []string {
	t.Helper()

	rows, err := db.QueryContext(testutil.Context(t), `SELECT step FROM migration_order ORDER BY rowid ASC`)
	if err != nil {
		t.Fatalf("QueryContext(migration_order) error = %v", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	steps := make([]string, 0)
	for rows.Next() {
		var step string
		if err := rows.Scan(&step); err != nil {
			t.Fatalf("Scan(step) error = %v", err)
		}
		steps = append(steps, step)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows.Err() = %v", err)
	}
	return steps
}

func nilMigrationContext() context.Context {
	return nil
}
