# Migration Test Patterns

## Required tests

### `Test<Name>FreshDB`

```go
t.Run("Should apply migration on fresh DB", func(t *testing.T) {
    t.Parallel()
    db := openInMemorySQLite(t) // or t.TempDir-backed file
    if err := store.RunMigrations(ctx, db, migrationsThroughNNN); err != nil {
        t.Fatalf("migrate: %v", err)
    }
    assertColumn(t, db, "foo", "new_column", "TEXT", true /* not null */)
    assertIndex(t, db, "idx_foo_new_column")
    assertSchemaMigrationsHas(t, db, NNN)
})
```

### `Test<Name>ReopenAfterRestart`

```go
t.Run("Should preserve data when migrating an existing DB", func(t *testing.T) {
    t.Parallel()
    dbPath := filepath.Join(t.TempDir(), "test.db")
    db := openSQLite(t, dbPath)

    // Apply only migrations up to NNN-1
    if err := store.RunMigrations(ctx, db, migrationsThrough(NNN-1)); err != nil {
        t.Fatalf("baseline migrate: %v", err)
    }
    // Insert legacy data
    if _, err := db.Exec(`INSERT INTO foo (id, legacy_value) VALUES (?, ?)`, 1, "abc"); err != nil {
        t.Fatalf("seed: %v", err)
    }
    db.Close()

    // Reopen and apply migration NNN
    db = openSQLite(t, dbPath)
    if err := store.RunMigrations(ctx, db, migrationsThroughNNN); err != nil {
        t.Fatalf("upgrade migrate: %v", err)
    }
    var got string
    if err := db.QueryRow(`SELECT new_column FROM foo WHERE id = 1`).Scan(&got); err != nil {
        t.Fatalf("read: %v", err)
    }
    if got != "abc" {
        t.Fatalf("backfill mismatch: got %q want %q", got, "abc")
    }
})
```

### `Test<Name>Idempotence`

```go
t.Run("Should be safe to apply twice if registry retries", func(t *testing.T) {
    t.Parallel()
    db := openInMemorySQLite(t)
    if err := store.RunMigrations(ctx, db, migrationsThroughNNN); err != nil {
        t.Fatalf("first run: %v", err)
    }
    if err := store.RunMigrations(ctx, db, migrationsThroughNNN); err != nil {
        t.Fatalf("second run: %v", err)
    }
    assertSchemaMigrationsCount(t, db, NNN, 1) // recorded once
})
```

### `Test<Name>RecoveryWalShm` (when touching recovery)

```go
t.Run("Should rename .db, -wal, and -shm during recovery", func(t *testing.T) {
    t.Parallel()
    dir := t.TempDir()
    base := filepath.Join(dir, "agh.db")
    seedSQLiteWithWALAndSHM(t, base)
    if err := store.RecoverSQLiteDatabase(base); err != nil {
        t.Fatalf("recover: %v", err)
    }
    for _, suffix := range []string{"", "-wal", "-shm"} {
        backup := base + suffix + ".bak"
        if _, err := os.Stat(backup); err != nil {
            t.Fatalf("expected %s; got %v", backup, err)
        }
    }
})
```

## Anti-patterns

- Tests that assert only the schema by string-matching `sqlite_master`.sql — that's fragile across SQLite versions. Use `pragma_table_info` and `pragma_index_list`.
- Tests that share a single in-memory DB across subtests (parallel-unsafe).
- Tests that assume `EnsureSchema` will "catch" missing schema — that's the bug this skill exists to prevent.
