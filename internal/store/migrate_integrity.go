package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"strconv"
	"strings"

	atlasmigrate "ariga.io/atlas/sql/migrate"
)

const migrationFamilyRecoveryInstruction = "stop AGH; preserve or move the complete AGH_HOME or workspace .agh " +
	"directory containing this database, including every sibling database file; then start AGH with a separately " +
	"selected fresh AGH_HOME or workspace"

type migrationDirectory struct {
	fsys       fs.FS
	maxVersion int64
	sumDigest  string
}

func loadMigrationDirectory(stream MigrationStream) (migrationDirectory, error) {
	directory, err := fs.Sub(stream.FS, stream.Dir)
	if err != nil {
		return migrationDirectory{}, fmt.Errorf("store: open migration stream %q directory: %w", stream.Name, err)
	}
	memoryDirectory := &atlasmigrate.MemDir{}
	maxVersion := int64(0)
	if err := fs.WalkDir(directory, ".", func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		contents, err := fs.ReadFile(directory, path)
		if err != nil {
			return fmt.Errorf("read %s: %w", path, err)
		}
		if err := memoryDirectory.WriteFile(filepath.ToSlash(path), contents); err != nil {
			return fmt.Errorf("copy %s: %w", path, err)
		}
		if filepath.Ext(path) == ".sql" {
			version, err := migrationFileVersion(path)
			if err != nil {
				return err
			}
			if version > maxVersion {
				maxVersion = version
			}
		}
		return nil
	}); err != nil {
		return migrationDirectory{}, fmt.Errorf("store: read migration stream %q: %w", stream.Name, err)
	}
	if maxVersion == 0 {
		return migrationDirectory{}, fmt.Errorf("store: migration stream %q has no SQL migrations", stream.Name)
	}
	if err := atlasmigrate.Validate(memoryDirectory); err != nil {
		return migrationDirectory{}, fmt.Errorf("store: validate migration stream %q atlas.sum: %w", stream.Name, err)
	}
	sumBytes, err := fs.ReadFile(directory, atlasmigrate.HashFileName)
	if err != nil {
		return migrationDirectory{}, fmt.Errorf("store: read migration stream %q atlas.sum: %w", stream.Name, err)
	}
	var hash atlasmigrate.HashFile
	if err := hash.UnmarshalText(sumBytes); err != nil {
		return migrationDirectory{}, fmt.Errorf("store: parse migration stream %q atlas.sum: %w", stream.Name, err)
	}
	return migrationDirectory{fsys: directory, maxVersion: maxVersion, sumDigest: hash.Sum()}, nil
}

func migrationFileVersion(path string) (int64, error) {
	base := filepath.Base(path)
	separator := strings.IndexByte(base, '_')
	if separator <= 0 {
		return 0, fmt.Errorf("store: invalid migration filename %q", base)
	}
	version, err := strconv.ParseInt(base[:separator], 10, 64)
	if err != nil || version <= 0 {
		return 0, fmt.Errorf("store: invalid migration filename %q", base)
	}
	return version, nil
}

func validateMigrationTableName(name string) error {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return errors.New("store: migration table name is required")
	}
	for idx, character := range trimmed {
		valid := character == '_' ||
			(character >= 'a' && character <= 'z') ||
			(character >= 'A' && character <= 'Z') ||
			(idx > 0 && character >= '0' && character <= '9')
		if !valid {
			return fmt.Errorf("store: invalid migration table name %q", trimmed)
		}
	}
	return nil
}

func refuseLegacyDatabase(ctx context.Context, db *sql.DB, stream MigrationStream, path string) error {
	for _, table := range stream.LegacyTables {
		exists, err := migrationTableExists(ctx, db, table)
		if err != nil {
			return err
		}
		if exists {
			return fmt.Errorf(
				"%w: %s contains legacy table %s; %s",
				ErrLegacyDatabase,
				path,
				table,
				migrationFamilyRecoveryInstruction,
			)
		}
	}
	return nil
}

func refuseAheadDatabase(
	ctx context.Context,
	db *sql.DB,
	stream MigrationStream,
	path string,
	maxEmbeddedVersion int64,
) error {
	status, err := readStreamStatus(ctx, db, stream, "")
	if err != nil {
		return err
	}
	if status.Version > maxEmbeddedVersion {
		return fmt.Errorf(
			"%w: %s stream %s is at version %d, binary head is %d; "+
				"install a newer AGH binary or %s",
			ErrSchemaAhead,
			path,
			stream.Name,
			status.Version,
			maxEmbeddedVersion,
			migrationFamilyRecoveryInstruction,
		)
	}
	return nil
}

func readStreamStatus(
	ctx context.Context,
	db *sql.DB,
	stream MigrationStream,
	sumDigest string,
) (status StreamStatus, err error) {
	status = StreamStatus{Stream: stream.Name, SumDigest: sumDigest}
	exists, err := migrationTableExists(ctx, db, stream.VersionTable)
	if err != nil {
		return StreamStatus{}, err
	}
	if !exists {
		return status, nil
	}
	rows, err := db.QueryContext(ctx, fmt.Sprintf(
		`SELECT version_id, is_applied FROM %s ORDER BY id ASC`,
		quoteIdentifier(stream.VersionTable),
	))
	if err != nil {
		return StreamStatus{}, fmt.Errorf("store: query migration stream %q status: %w", stream.Name, err)
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			closeErr = fmt.Errorf("store: close migration stream %q status rows: %w", stream.Name, closeErr)
			if err == nil {
				err = closeErr
				return
			}
			err = errors.Join(err, closeErr)
		}
	}()
	applied := make(map[int64]bool)
	for rows.Next() {
		var version int64
		var isApplied bool
		if err := rows.Scan(&version, &isApplied); err != nil {
			return StreamStatus{}, fmt.Errorf("store: scan migration stream %q status: %w", stream.Name, err)
		}
		if version > 0 {
			applied[version] = isApplied
		}
	}
	if err := rows.Err(); err != nil {
		return StreamStatus{}, fmt.Errorf("store: iterate migration stream %q status: %w", stream.Name, err)
	}
	for version, isApplied := range applied {
		if !isApplied {
			continue
		}
		status.AppliedCount++
		if version > status.Version {
			status.Version = version
		}
	}
	return status, nil
}

func migrationTableExists(ctx context.Context, db *sql.DB, table string) (bool, error) {
	var count int
	if err := db.QueryRowContext(
		ctx,
		`SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = ?`,
		table,
	).Scan(&count); err != nil {
		return false, fmt.Errorf("store: query migration table %q: %w", table, err)
	}
	return count > 0, nil
}

func quoteIdentifier(name string) string {
	return `"` + name + `"`
}

func migrationDatabasePath(ctx context.Context, db *sql.DB) (path string, err error) {
	rows, err := db.QueryContext(ctx, "PRAGMA database_list")
	if err != nil {
		return "", fmt.Errorf("store: query migration database path: %w", err)
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			closeErr = fmt.Errorf("store: close migration database path rows: %w", closeErr)
			if err == nil {
				err = closeErr
				return
			}
			err = errors.Join(err, closeErr)
		}
	}()
	for rows.Next() {
		var sequence int
		var name string
		var filename string
		if err := rows.Scan(&sequence, &name, &filename); err != nil {
			return "", fmt.Errorf("store: scan migration database path: %w", err)
		}
		if name == "main" {
			if filename == "" {
				return "in-memory database", nil
			}
			return filename, nil
		}
	}
	if err := rows.Err(); err != nil {
		return "", fmt.Errorf("store: iterate migration database path: %w", err)
	}
	return "unknown database", nil
}
