package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"time"

	"github.com/pressly/goose/v3"
)

var (
	// ErrLegacyDatabase reports a database created by the removed migration runner.
	ErrLegacyDatabase = errors.New("store: database was created by the removed pre-goose migration runner")
	// ErrSchemaAhead reports a database newer than the running binary.
	ErrSchemaAhead = errors.New("store: database schema is ahead of this binary")
)

// MigrationStream describes one independently versioned embedded schema.
type MigrationStream struct {
	Name         string
	FS           fs.FS
	Dir          string
	VersionTable string
	LegacyTables []string
}

// StreamStatus describes the applied state of one migration stream.
type StreamStatus struct {
	Stream       string
	Version      int64
	AppliedCount int
	SumDigest    string
}

// Apply validates and applies all pending SQL migrations for stream.
func Apply(ctx context.Context, db *sql.DB, stream MigrationStream) error {
	if err := validateMigrationInputs(ctx, db, stream); err != nil {
		return err
	}
	path, err := migrationDatabasePath(ctx, db)
	if err != nil {
		return err
	}
	if err := refuseLegacyDatabase(ctx, db, stream, path); err != nil {
		migrationLogger(ctx).ErrorContext(ctx, "store.migrations.refused",
			"stream", stream.Name,
			"reason", "legacy_database",
			"path", path,
		)
		return err
	}
	directory, err := loadMigrationDirectory(stream)
	if err != nil {
		migrationLogger(ctx).ErrorContext(ctx, "store.migrations.refused",
			"stream", stream.Name,
			"reason", "sum_mismatch",
			"path", path,
		)
		return err
	}
	if err := refuseAheadDatabase(ctx, db, stream, path, directory.maxVersion); err != nil {
		return err
	}

	startedAt := time.Now()
	provider, err := goose.NewProvider(
		goose.DialectSQLite3,
		db,
		directory.fsys,
		goose.WithTableName(stream.VersionTable),
		goose.WithDisableGlobalRegistry(true),
	)
	if err != nil {
		return fmt.Errorf("store: create migration provider for stream %q: %w", stream.Name, err)
	}
	if _, err := provider.Up(ctx); err != nil {
		return fmt.Errorf("store: apply migration stream %q: %w", stream.Name, err)
	}
	status, err := readStreamStatus(ctx, db, stream, directory.sumDigest)
	if err != nil {
		return err
	}
	migrationLogger(ctx).InfoContext(ctx, "store.migrations.applied",
		"stream", status.Stream,
		"version", status.Version,
		"applied_count", status.AppliedCount,
		"duration_ms", time.Since(startedAt).Milliseconds(),
	)
	return nil
}

// Status returns stream state without creating or mutating a version table.
func Status(ctx context.Context, db *sql.DB, stream MigrationStream) (StreamStatus, error) {
	if err := validateMigrationInputs(ctx, db, stream); err != nil {
		return StreamStatus{}, err
	}
	path, err := migrationDatabasePath(ctx, db)
	if err != nil {
		return StreamStatus{}, err
	}
	if err := refuseLegacyDatabase(ctx, db, stream, path); err != nil {
		return StreamStatus{}, err
	}
	directory, err := loadMigrationDirectory(stream)
	if err != nil {
		return StreamStatus{}, err
	}
	if err := refuseAheadDatabase(ctx, db, stream, path, directory.maxVersion); err != nil {
		return StreamStatus{}, err
	}
	return readStreamStatus(ctx, db, stream, directory.sumDigest)
}

func validateMigrationInputs(ctx context.Context, db *sql.DB, stream MigrationStream) error {
	if ctx == nil {
		return errors.New("store: migration context is required")
	}
	if db == nil {
		return errors.New("store: migration database is required")
	}
	if stream.Name == "" {
		return errors.New("store: migration stream name is required")
	}
	if stream.FS == nil {
		return fmt.Errorf("store: migration stream %q filesystem is required", stream.Name)
	}
	if stream.Dir == "" {
		return fmt.Errorf("store: migration stream %q directory is required", stream.Name)
	}
	if err := validateMigrationTableName(stream.VersionTable); err != nil {
		return fmt.Errorf("store: migration stream %q version table: %w", stream.Name, err)
	}
	for _, table := range stream.LegacyTables {
		if err := validateMigrationTableName(table); err != nil {
			return fmt.Errorf("store: migration stream %q legacy table: %w", stream.Name, err)
		}
	}
	return nil
}
