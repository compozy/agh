package store

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"

	"github.com/compozy/agh/internal/testutil"
)

func TestOpenSQLiteDatabaseRecoveryContract(t *testing.T) {
	t.Parallel()

	t.Run("Should not quarantine healthy database after malformed initialization error", func(t *testing.T) {
		t.Parallel()

		ctx := testutil.Context(t)
		dbPath := filepath.Join(t.TempDir(), "healthy.db")
		db, err := OpenSQLiteDatabase(ctx, dbPath, func(ctx context.Context, db *sql.DB) error {
			return EnsureSchema(ctx, db, []string{
				`CREATE TABLE IF NOT EXISTS sentinel (id TEXT PRIMARY KEY, value TEXT NOT NULL);`,
				`INSERT INTO sentinel (id, value) VALUES ('row-1', 'alpha');`,
			})
		})
		if err != nil {
			t.Fatalf("OpenSQLiteDatabase(seed) error = %v", err)
		}
		if err := db.Close(); err != nil {
			t.Fatalf("Close(seed) error = %v", err)
		}

		initErr := errors.New("malformed config")
		db, err = OpenSQLiteDatabase(ctx, dbPath, func(context.Context, *sql.DB) error {
			return initErr
		})
		if err == nil {
			if db != nil {
				if closeErr := db.Close(); closeErr != nil {
					t.Fatalf("Close(unexpected) error = %v", closeErr)
				}
			}
			t.Fatal("OpenSQLiteDatabase(init fail) error = nil, want initialization error")
		}
		if !errors.Is(err, initErr) {
			t.Fatalf("OpenSQLiteDatabase(init fail) error = %v, want %v", err, initErr)
		}
		matches, err := filepath.Glob(dbPath + ".corrupt.*")
		if err != nil {
			t.Fatalf("Glob(corrupt files) error = %v", err)
		}
		if len(matches) != 0 {
			t.Fatalf("corrupt quarantine files = %v, want none", matches)
		}

		reopened, err := OpenSQLiteDatabase(ctx, dbPath, nil)
		if err != nil {
			t.Fatalf("OpenSQLiteDatabase(reopen) error = %v", err)
		}
		t.Cleanup(func() {
			if closeErr := reopened.Close(); closeErr != nil {
				t.Errorf("Close(reopen) error = %v", closeErr)
			}
		})

		var value string
		if err := reopened.QueryRowContext(ctx, `SELECT value FROM sentinel WHERE id = 'row-1'`).
			Scan(&value); err != nil {
			t.Fatalf("QueryRowContext(sentinel) error = %v", err)
		}
		if value != "alpha" {
			t.Fatalf("sentinel value = %q, want alpha", value)
		}
	})
}
