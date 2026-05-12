# Migration Template

Replace the placeholders. Match the project's existing migration registration shape (Go function or registered SQL).

## Header

```go
// Migration NNN: <one-line summary>
//
// Why: <root cause / techspec section reference / ADR reference>
// Affects: <database name> table(s) <list>
// Idempotent: <yes/no>; <if no, why>
// Reversible: <yes/no>; <if yes, paired down migration NNN_down>
```

## Body skeleton

```go
func migrationNNN(ctx context.Context, tx *sql.Tx) error {
    // 1. Guard against re-application (defense in depth; the registry already prevents double-apply).
    var exists int
    if err := tx.QueryRowContext(ctx,
        `SELECT COUNT(*) FROM pragma_table_info('foo') WHERE name = 'new_column'`).Scan(&exists); err != nil {
        return fmt.Errorf("migration NNN: probe table_info: %w", err)
    }
    if exists > 0 {
        return nil // already applied via prior partial run
    }

    // 2. Schema change.
    if _, err := tx.ExecContext(ctx,
        `ALTER TABLE foo ADD COLUMN new_column TEXT NOT NULL DEFAULT ''`); err != nil {
        return fmt.Errorf("migration NNN: add column: %w", err)
    }

    // 3. Backfill (if needed).
    if _, err := tx.ExecContext(ctx,
        `UPDATE foo SET new_column = COALESCE(legacy_value, '') WHERE new_column = ''`); err != nil {
        return fmt.Errorf("migration NNN: backfill: %w", err)
    }

    // 4. Index (if needed).
    if _, err := tx.ExecContext(ctx,
        `CREATE INDEX IF NOT EXISTS idx_foo_new_column ON foo(new_column)`); err != nil {
        return fmt.Errorf("migration NNN: create index: %w", err)
    }

    // 5. Record applied migration.
    if _, err := tx.ExecContext(ctx,
        `INSERT INTO schema_migrations (version, applied_at) VALUES (?, ?)`,
        NNN, time.Now().UTC().Format(time.RFC3339Nano)); err != nil {
        return fmt.Errorf("migration NNN: record: %w", err)
    }

    return nil
}
```

## Transactional wrap

The migration runner wraps each migration in `BEGIN IMMEDIATE` for SQLite. The function above receives the `*sql.Tx`. Do not begin/commit inside.

## Column rename pattern

```go
// Pattern: CREATE NEW + INSERT INTO ... SELECT + DROP + RENAME
//
// 1. CREATE TABLE foo_new (... new_column TEXT NOT NULL ...)
// 2. INSERT INTO foo_new (cols) SELECT cols (mapping legacy_column → new_column) FROM foo
// 3. DROP TABLE foo
// 4. ALTER TABLE foo_new RENAME TO foo
// 5. Recreate indices on foo
//
// This costs O(n) but is the only safe rename pattern below SQLite 3.35.
```
