package store

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

const schemaMigrationsTable = "schema_migrations"

type migrationConfig struct {
	table string
}

// MigrationOption customizes migration execution.
type MigrationOption func(*migrationConfig)

// WithMigrationsTable stores migration records in a subsystem-specific table.
// Use this when independent migration streams share one SQLite database file.
func WithMigrationsTable(name string) MigrationOption {
	return func(cfg *migrationConfig) {
		if cfg == nil {
			return
		}
		cfg.table = name
	}
}

// Migration describes one ordered SQLite schema change.
type Migration struct {
	Version    int
	Name       string
	Statements []string
	Up         func(ctx context.Context, tx *sql.Tx) error
	Checksum   string
}

// MigrationRecord describes one applied schema migration row.
type MigrationRecord struct {
	Version   int
	Name      string
	Checksum  string
	AppliedAt time.Time
}

// EnsureSchema executes each schema statement in order.
func EnsureSchema(ctx context.Context, db *sql.DB, statements []string) error {
	for _, stmt := range statements {
		if _, err := db.ExecContext(ctx, stmt); err != nil {
			return err
		}
	}
	return nil
}

// RunMigrations applies pending migrations once in deterministic version order.
func RunMigrations(ctx context.Context, db *sql.DB, migrations []Migration, opts ...MigrationOption) error {
	if ctx == nil {
		return errors.New("store: migrate schema context is required")
	}
	if db == nil {
		return errors.New("store: migrate schema database is required")
	}
	cfg, err := newMigrationConfig(opts...)
	if err != nil {
		return err
	}
	ordered, err := normalizeMigrations(migrations)
	if err != nil {
		return err
	}
	if err := ensureSchemaMigrationsTable(ctx, db, cfg.table); err != nil {
		return err
	}

	applied, err := appliedMigrationRecords(ctx, db, cfg.table)
	if err != nil {
		return err
	}
	for _, migration := range ordered {
		checksum, err := migrationChecksum(migration)
		if err != nil {
			return err
		}
		if record, ok := applied[migration.Version]; ok {
			if record.Name != strings.TrimSpace(migration.Name) || record.Checksum != checksum {
				return fmt.Errorf(
					"store: migration %d integrity mismatch: recorded %q/%s, current %q/%s",
					migration.Version,
					record.Name,
					record.Checksum,
					strings.TrimSpace(migration.Name),
					checksum,
				)
			}
			continue
		}
		if err := applyMigration(ctx, db, cfg.table, migration, checksum); err != nil {
			return err
		}
	}
	return nil
}

// MigrationChecksum returns the checksum RunMigrations records for migration.
func MigrationChecksum(migration Migration) (string, error) {
	return migrationChecksum(migration)
}

// AppliedMigrations returns applied migration records ordered by version.
func AppliedMigrations(ctx context.Context, db *sql.DB) ([]MigrationRecord, error) {
	return appliedMigrations(ctx, db, schemaMigrationsTable)
}

func newMigrationConfig(opts ...MigrationOption) (migrationConfig, error) {
	cfg := migrationConfig{table: schemaMigrationsTable}
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}
	table, err := normalizeMigrationTableName(cfg.table)
	if err != nil {
		return migrationConfig{}, err
	}
	cfg.table = table
	return cfg, nil
}

func normalizeMigrationTableName(name string) (string, error) {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return "", errors.New("store: migration table name is required")
	}
	for idx, r := range trimmed {
		valid := r == '_' ||
			(r >= 'a' && r <= 'z') ||
			(r >= 'A' && r <= 'Z') ||
			(idx > 0 && r >= '0' && r <= '9')
		if !valid {
			return "", fmt.Errorf("store: invalid migration table name %q", trimmed)
		}
	}
	return trimmed, nil
}

func quoteIdentifier(name string) string {
	return `"` + name + `"`
}

func normalizeMigrations(migrations []Migration) ([]Migration, error) {
	ordered := append([]Migration(nil), migrations...)
	sort.SliceStable(ordered, func(i, j int) bool {
		if ordered[i].Version == ordered[j].Version {
			return strings.TrimSpace(ordered[i].Name) < strings.TrimSpace(ordered[j].Name)
		}
		return ordered[i].Version < ordered[j].Version
	})

	versions := make(map[int]string, len(ordered))
	names := make(map[string]int, len(ordered))
	for _, migration := range ordered {
		name := strings.TrimSpace(migration.Name)
		if migration.Version <= 0 {
			return nil, fmt.Errorf("store: migration %q has invalid version %d", name, migration.Version)
		}
		if name == "" {
			return nil, fmt.Errorf("store: migration %d name is required", migration.Version)
		}
		if previous, ok := versions[migration.Version]; ok {
			return nil, fmt.Errorf(
				"store: duplicate migration version %d for %q and %q",
				migration.Version,
				previous,
				name,
			)
		}
		if previous, ok := names[name]; ok {
			return nil, fmt.Errorf(
				"store: duplicate migration name %q for versions %d and %d",
				name,
				previous,
				migration.Version,
			)
		}
		if migration.Up == nil && len(migration.Statements) == 0 {
			return nil, fmt.Errorf("store: migration %d %q has no operation", migration.Version, name)
		}
		versions[migration.Version] = name
		names[name] = migration.Version
	}
	return ordered, nil
}

func ensureSchemaMigrationsTable(ctx context.Context, db *sql.DB, table string) (err error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("store: begin schema migrations bootstrap: %w", err)
	}
	defer rollbackMigrationTx(&err, tx, "schema migrations bootstrap")

	quotedTable := quoteIdentifier(table)
	if _, err := tx.ExecContext(ctx, fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
		version    INTEGER PRIMARY KEY,
		name       TEXT NOT NULL,
		checksum   TEXT NOT NULL,
		applied_at TEXT NOT NULL
	);`, quotedTable)); err != nil {
		return fmt.Errorf("store: create schema_migrations table: %w", err)
	}
	indexName := quoteIdentifier("idx_" + table + "_name")
	if _, err := tx.ExecContext(ctx, fmt.Sprintf(`
		CREATE UNIQUE INDEX IF NOT EXISTS %s
		ON %s(name);
	`, indexName, quotedTable)); err != nil {
		return fmt.Errorf("store: create schema_migrations name index: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("store: commit schema migrations bootstrap: %w", err)
	}
	return nil
}

func appliedMigrationRecords(ctx context.Context, db *sql.DB, table string) (map[int]MigrationRecord, error) {
	records, err := appliedMigrations(ctx, db, table)
	if err != nil {
		return nil, err
	}
	applied := make(map[int]MigrationRecord, len(records))
	for _, record := range records {
		applied[record.Version] = record
	}
	return applied, nil
}

func applyMigration(ctx context.Context, db *sql.DB, table string, migration Migration, checksum string) (err error) {
	name := strings.TrimSpace(migration.Name)
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("store: begin migration %d %q: %w", migration.Version, name, err)
	}
	defer rollbackMigrationTx(&err, tx, fmt.Sprintf("migration %d %q", migration.Version, name))

	if migration.Up != nil {
		if err := migration.Up(ctx, tx); err != nil {
			return fmt.Errorf("store: apply migration %d %q: %w", migration.Version, name, err)
		}
	} else {
		for _, statement := range migration.Statements {
			if err := ctx.Err(); err != nil {
				return fmt.Errorf("store: apply migration %d %q: %w", migration.Version, name, err)
			}
			if _, err := tx.ExecContext(ctx, statement); err != nil {
				return fmt.Errorf("store: apply migration %d %q: %w", migration.Version, name, err)
			}
		}
	}

	if _, err := tx.ExecContext(
		ctx,
		fmt.Sprintf(`INSERT INTO %s (version, name, checksum, applied_at) VALUES (?, ?, ?, ?)`, quoteIdentifier(table)),
		migration.Version,
		name,
		checksum,
		FormatTimestamp(time.Now().UTC()),
	); err != nil {
		return fmt.Errorf("store: record migration %d %q: %w", migration.Version, name, err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("store: commit migration %d %q: %w", migration.Version, name, err)
	}
	return nil
}

func migrationChecksum(migration Migration) (string, error) {
	if checksum := strings.TrimSpace(migration.Checksum); checksum != "" {
		return checksum, nil
	}
	name := strings.TrimSpace(migration.Name)
	if len(migration.Statements) == 0 {
		return "", fmt.Errorf(
			"store: migration %d %q checksum is required for custom operation",
			migration.Version,
			name,
		)
	}

	hash := sha256.New()
	if _, err := fmt.Fprintf(hash, "%d\n%s\n", migration.Version, name); err != nil {
		return "", fmt.Errorf(
			"store: hash migration %d %q header: %w",
			migration.Version,
			name,
			err,
		)
	}
	for _, statement := range migration.Statements {
		if _, err := hash.Write([]byte(strings.TrimSpace(statement))); err != nil {
			return "", fmt.Errorf(
				"store: hash migration %d %q statement: %w",
				migration.Version,
				name,
				err,
			)
		}
		if _, err := hash.Write([]byte{'\n'}); err != nil {
			return "", fmt.Errorf(
				"store: hash migration %d %q separator: %w",
				migration.Version,
				name,
				err,
			)
		}
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func appliedMigrations(ctx context.Context, db *sql.DB, table string) ([]MigrationRecord, error) {
	if ctx == nil {
		return nil, errors.New("store: list schema migrations context is required")
	}
	if db == nil {
		return nil, errors.New("store: list schema migrations database is required")
	}
	normalizedTable, err := normalizeMigrationTableName(table)
	if err != nil {
		return nil, err
	}
	exists, err := migrationTableExists(ctx, db, normalizedTable)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, nil
	}

	rows, err := db.QueryContext(
		ctx,
		fmt.Sprintf(`
		SELECT version, name, checksum, applied_at
		FROM %s
		ORDER BY version ASC
	`, quoteIdentifier(normalizedTable)),
	)
	if err != nil {
		return nil, fmt.Errorf("store: query schema migrations: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	records := make([]MigrationRecord, 0)
	for rows.Next() {
		var (
			record     MigrationRecord
			appliedRaw string
		)
		if err := rows.Scan(&record.Version, &record.Name, &record.Checksum, &appliedRaw); err != nil {
			return nil, fmt.Errorf("store: scan schema migration: %w", err)
		}
		appliedAt, err := ParseTimestamp(appliedRaw)
		if err != nil {
			return nil, fmt.Errorf("store: parse schema migration timestamp: %w", err)
		}
		record.AppliedAt = appliedAt
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: iterate schema migrations: %w", err)
	}
	return records, nil
}

func migrationTableExists(ctx context.Context, db *sql.DB, table string) (bool, error) {
	var count int
	if err := db.QueryRowContext(
		ctx,
		`SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = ?`,
		table,
	).Scan(&count); err != nil {
		return false, fmt.Errorf("store: query schema_migrations table: %w", err)
	}
	return count > 0, nil
}

func rollbackMigrationTx(target *error, tx *sql.Tx, action string) {
	if tx == nil || target == nil {
		return
	}
	if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
		rollbackErr := fmt.Errorf("store: rollback %s: %w", action, err)
		if *target == nil {
			*target = rollbackErr
			return
		}
		*target = errors.Join(*target, rollbackErr)
	}
}
