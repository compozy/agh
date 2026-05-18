// Package workspacedb owns per-workspace SQLite database lifecycle helpers.
package workspacedb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"sync/atomic"

	"github.com/pedronauck/agh/internal/store"
	aghworkspace "github.com/pedronauck/agh/internal/workspace"
)

const defaultMigrationTable = "schema_migrations"

var errAheadSchema = errors.New("store: workspace database schema ahead of binary")

// DB is an open per-workspace AGH database handle.
type DB struct {
	db            *sql.DB
	path          string
	workspaceRoot string
	identity      aghworkspace.Identity
	closed        atomic.Int32
}

// Options configures a per-workspace database open.
type Options struct {
	WorkspaceRoot   string
	Migrations      []store.Migration
	MigrationsTable string
}

// Open resolves the workspace identity and opens <workspace>/.agh/agh.db.
func Open(ctx context.Context, opts Options) (*DB, error) {
	if ctx == nil {
		return nil, errors.New("store: open workspace database context is required")
	}
	workspaceRoot := strings.TrimSpace(opts.WorkspaceRoot)
	if workspaceRoot == "" {
		return nil, errors.New("store: workspace root is required")
	}

	identity, err := aghworkspace.EnsureIdentity(ctx, workspaceRoot)
	if err != nil {
		return nil, fmt.Errorf("store: resolve workspace identity for %q: %w", workspaceRoot, err)
	}
	dbPath := filepath.Join(filepath.Dir(identity.Path), store.GlobalDatabaseName)
	migrationTable := normalizeMigrationTable(opts.MigrationsTable)
	db, err := store.OpenSQLiteDatabase(ctx, dbPath, func(ctx context.Context, db *sql.DB) error {
		return runWorkspaceMigrations(ctx, db, opts.Migrations, migrationTable)
	})
	if err != nil {
		return nil, fmt.Errorf("store: open workspace database %q: %w", dbPath, err)
	}

	return &DB{
		db:            db,
		path:          dbPath,
		workspaceRoot: workspaceRoot,
		identity:      identity,
	}, nil
}

// OpenWorkspace opens a workspace database with the default migration table.
func OpenWorkspace(ctx context.Context, workspaceRoot string, migrations []store.Migration) (*DB, error) {
	return Open(ctx, Options{
		WorkspaceRoot: workspaceRoot,
		Migrations:    migrations,
	})
}

// Path reports the database path.
func (d *DB) Path() string {
	if d == nil {
		return ""
	}
	return d.path
}

// WorkspaceID reports the resolved workspace identity.
func (d *DB) WorkspaceID() string {
	if d == nil {
		return ""
	}
	return d.identity.WorkspaceID
}

// WorkspaceRoot reports the workspace root used to open the database.
func (d *DB) WorkspaceRoot() string {
	if d == nil {
		return ""
	}
	return d.workspaceRoot
}

// DB exposes the underlying SQL handle for storage packages.
func (d *DB) DB() *sql.DB {
	if d == nil {
		return nil
	}
	return d.db
}

// Close checkpoints the WAL and closes the database.
func (d *DB) Close(ctx context.Context) error {
	if d == nil {
		return nil
	}
	if ctx == nil {
		return errors.New("store: close workspace database context is required")
	}
	if !d.closed.CompareAndSwap(0, 1) {
		return nil
	}

	checkpointErr := store.Checkpoint(ctx, d.db)
	closeErr := d.db.Close()
	return errors.Join(checkpointErr, closeErr)
}

func runWorkspaceMigrations(
	ctx context.Context,
	db *sql.DB,
	migrations []store.Migration,
	migrationTable string,
) error {
	if err := rejectAheadSchema(ctx, db, migrations, migrationTable); err != nil {
		return err
	}
	if err := store.RunMigrations(
		ctx,
		db,
		migrations,
		store.WithMigrationsTable(migrationTable),
	); err != nil {
		return err
	}
	return rejectAheadSchema(ctx, db, migrations, migrationTable)
}

func rejectAheadSchema(
	ctx context.Context,
	db *sql.DB,
	migrations []store.Migration,
	migrationTable string,
) error {
	records, err := store.AppliedMigrationsWithTable(ctx, db, migrationTable)
	if err != nil {
		return err
	}
	head := migrationHead(migrations)
	for _, record := range records {
		if record.Version > head {
			return fmt.Errorf(
				"%w: workspace database schema version %d is ahead of binary head %d",
				errAheadSchema,
				record.Version,
				head,
			)
		}
	}
	return nil
}

func migrationHead(migrations []store.Migration) int {
	head := 0
	for _, migration := range migrations {
		if migration.Version > head {
			head = migration.Version
		}
	}
	return head
}

func normalizeMigrationTable(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return defaultMigrationTable
	}
	return trimmed
}
