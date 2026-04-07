package store

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/pedronauck/agh/internal/testutil"
)

func TestStoreSQLHelpers(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 4, 12, 0, 0, 0, time.UTC)
	where, args := BuildClauses(
		StringClause("type", " agent_message "),
		StringClause("ignored", "   "),
		TimeClause("timestamp", ">=", now),
		TimeClause("timestamp", ">=", time.Time{}),
		Int64Clause("sequence", ">", 3),
		Int64Clause("sequence", ">", 0),
	)

	if got, want := NormalizeSessionType("   "), defaultSessionType; got != want {
		t.Fatalf("NormalizeSessionType(blank) = %q, want %q", got, want)
	}
	if got := NormalizeSessionType(" dream "); got != "dream" {
		t.Fatalf("NormalizeSessionType(value) = %q, want dream", got)
	}

	if got, want := len(where), 3; got != want {
		t.Fatalf("len(where) = %d, want %d (%v)", got, want, where)
	}
	if got, want := len(args), 3; got != want {
		t.Fatalf("len(args) = %d, want %d (%v)", got, want, args)
	}

	query := AppendWhere("SELECT * FROM events", where)
	if !strings.Contains(query, "WHERE type = ? AND timestamp >= ? AND sequence > ?") {
		t.Fatalf("AppendWhere() = %q", query)
	}

	invalidWhere, invalidArgs := BuildClauses(
		StringClause("bad-name", "value"),
		TimeClause("timestamp", "DROP TABLE", now),
		Int64Clause("sequence", "DROP TABLE", 3),
	)
	if got, want := invalidWhere, []string{"1 = 0", "1 = 0", "1 = 0"}; !testutil.EqualStringSlices(got, want) {
		t.Fatalf("invalid where = %#v, want %#v", got, want)
	}
	if got, want := len(invalidArgs), 0; got != want {
		t.Fatalf("len(invalidArgs) = %d, want %d", got, want)
	}

	limitedQuery, limitedArgs := AppendLimit(query, args, 5)
	if !strings.HasSuffix(limitedQuery, " LIMIT ?") {
		t.Fatalf("AppendLimit() query = %q", limitedQuery)
	}
	if got, want := limitedArgs[len(limitedArgs)-1], any(5); got != want {
		t.Fatalf("AppendLimit() last arg = %#v, want %#v", got, want)
	}
	if got, want := AppendWhere("SELECT 1", nil), "SELECT 1"; got != want {
		t.Fatalf("AppendWhere(no clauses) = %q, want %q", got, want)
	}
	if gotQuery, gotArgs := AppendLimit("SELECT 1", nil, 0); gotQuery != "SELECT 1" || gotArgs != nil {
		t.Fatalf("AppendLimit(no limit) = (%q, %#v), want (%q, nil)", gotQuery, gotArgs, "SELECT 1")
	}
}

func TestStoreSQLiteHelpers(t *testing.T) {
	t.Parallel()

	if got, want := sqliteDSN("/tmp/example.db"), "file:///tmp/example.db"; got != want {
		t.Fatalf("sqliteDSN() = %q, want %q", got, want)
	}
	if got, want := NullableInt64(nil), any(nil); got != want {
		t.Fatalf("NullableInt64(nil) = %#v, want nil", got)
	}
	value := int64(7)
	if got := NullableInt64(&value); got != int64(7) {
		t.Fatalf("NullableInt64(valid) = %#v, want 7", got)
	}
	if got, want := NullableFloat64(nil), any(nil); got != want {
		t.Fatalf("NullableFloat64(nil) = %#v, want nil", got)
	}
	floatValue := 1.5
	if got := NullableFloat64(&floatValue); got != 1.5 {
		t.Fatalf("NullableFloat64(valid) = %#v, want 1.5", got)
	}
	if got := NullString(sql.NullString{String: "   ", Valid: true}); got != nil {
		t.Fatalf("NullString(blank) = %#v, want nil", got)
	}
	if _, err := NormalizeSQLiteIdentifier("bad-name"); err == nil {
		t.Fatal("NormalizeSQLiteIdentifier(invalid) error = nil, want non-nil")
	}
	if got, err := NormalizeSQLiteIdentifier("valid_name_2"); err != nil || got != "valid_name_2" {
		t.Fatalf("NormalizeSQLiteIdentifier(valid) = (%q, %v), want (valid_name_2, nil)", got, err)
	}

	dbPath := filepath.Join(t.TempDir(), "shared.db")
	db, err := openSQLiteDatabaseOnce(testutil.Context(t), dbPath, func(ctx context.Context, db *sql.DB) error {
		return EnsureSchema(ctx, db, []string{
			`CREATE TABLE IF NOT EXISTS sample (id TEXT PRIMARY KEY, value TEXT NOT NULL);`,
			`INSERT INTO sample (id, value) VALUES ('row-1', 'alpha');`,
		})
	})
	if err != nil {
		t.Fatalf("openSQLiteDatabaseOnce() error = %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if err := configureSQLite(testutil.Context(t), db); err != nil {
		t.Fatalf("configureSQLite() error = %v", err)
	}
	if err := Checkpoint(testutil.Context(t), db); err != nil {
		t.Fatalf("Checkpoint() error = %v", err)
	}

	if mode, err := querySingleString(testutil.Context(t), db, "PRAGMA journal_mode"); err != nil || !strings.EqualFold(mode, "wal") {
		t.Fatalf("querySingleString(journal_mode) = (%q, %v), want wal", mode, err)
	}

	var count int
	if err := db.QueryRowContext(testutil.Context(t), `SELECT COUNT(*) FROM sample`).Scan(&count); err != nil {
		t.Fatalf("QueryRowContext(count) error = %v", err)
	}
	if count != 1 {
		t.Fatalf("sample row count = %d, want 1", count)
	}
}

func TestStoreSQLiteRecoveryAndFailures(t *testing.T) {
	t.Parallel()

	dbPath := filepath.Join(t.TempDir(), "recover.db")
	if err := os.WriteFile(dbPath, []byte("not a sqlite database"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	db, err := OpenSQLiteDatabase(testutil.Context(t), dbPath, func(ctx context.Context, db *sql.DB) error {
		return EnsureSchema(ctx, db, []string{`CREATE TABLE IF NOT EXISTS recovered (id TEXT PRIMARY KEY);`})
	})
	if err != nil {
		t.Fatalf("OpenSQLiteDatabase() error = %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	matches, err := filepath.Glob(dbPath + ".corrupt.*")
	if err != nil {
		t.Fatalf("Glob() error = %v", err)
	}
	if got, want := len(matches), 1; got != want {
		t.Fatalf("len(corrupt files) = %d, want %d (%v)", got, want, matches)
	}

	if _, err := openSQLiteDatabaseOnce(testutil.Context(t), filepath.Join(t.TempDir(), "init-fail.db"), func(ctx context.Context, db *sql.DB) error {
		return errors.New("boom")
	}); err == nil || !strings.Contains(err.Error(), "initialize sqlite database") {
		t.Fatalf("openSQLiteDatabaseOnce(init fail) error = %v, want initialize failure", err)
	}

	renamePath := filepath.Join(t.TempDir(), "rename.db")
	if err := os.WriteFile(renamePath, []byte("rename-me"), 0o644); err != nil {
		t.Fatalf("WriteFile(rename) error = %v", err)
	}
	for _, suffix := range []string{"-wal", "-shm"} {
		if err := os.WriteFile(renamePath+suffix, []byte("sidecar"), 0o644); err != nil {
			t.Fatalf("WriteFile(%s) error = %v", suffix, err)
		}
	}
	corruptPath, err := recoverSQLiteDatabase(renamePath)
	if err != nil {
		t.Fatalf("recoverSQLiteDatabase() error = %v", err)
	}
	if !strings.Contains(corruptPath, ".corrupt.") {
		t.Fatalf("recoverSQLiteDatabase() = %q, want .corrupt. suffix", corruptPath)
	}
	if _, err := os.Stat(corruptPath); err != nil {
		t.Fatalf("Stat(corruptPath) error = %v", err)
	}
	for _, suffix := range []string{"-wal", "-shm"} {
		if _, err := os.Stat(corruptPath + suffix); err != nil {
			t.Fatalf("Stat(%s) error = %v", corruptPath+suffix, err)
		}
	}
}
