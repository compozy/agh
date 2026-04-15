# GoClaw Store/SQLite Patterns — Analysis for AGH

## Key Findings

### 1. Per-Connection PRAGMA Wrapper (HIGH IMPACT)

GoClaw uses a `pragmaConnector` wrapper that applies PRAGMAs to **every new connection** — critical for concurrency since `db.Exec()` only applies to one connection.

```go
type pragmaConnector struct {
    driver  driver.Driver
    dsn     string
    pragmas []string
}

func (c *pragmaConnector) Connect(ctx context.Context) (driver.Conn, error) {
    conn, err := c.driver.Open(c.dsn)
    if err != nil { return nil, err }
    for _, p := range c.pragmas {
        // exec on conn (not db)
    }
    return conn, nil
}

// Usage: sql.OpenDB(&pragmaConnector{...})
```

PRAGMAs applied per-connection:

- `journal_mode = WAL` (concurrent readers)
- `busy_timeout = 15000` (15s before SQLITE_BUSY)
- `synchronous = NORMAL` (balance safety/performance)
- `cache_size = -8000` (8MB)
- `foreign_keys = ON`

Connection pool: 4 connections max (WAL allows 3 readers + 1 writer).

**AGH gap**: Uses query parameter pragmas in DSN string — simpler but less robust. Potential concurrency issues under load.

### 2. Embedded Schema + Migration Versioning (MEDIUM IMPACT)

```go
//go:embed schema.sql
var schemaSQL string

const SchemaVersion = 20

var migrations = map[int]string{
    1: `ALTER TABLE ...`,
    2: `CREATE TABLE ...`,
}

func EnsureSchema(db *sql.DB) error {
    // Fresh DB → apply schemaSQL + set version
    // Existing DB → apply patches v0→v1→v2...→vLatest
    // Idempotent master tenant seed
}
```

Backfill hooks for migrations needing Go logic (e.g., v15→v16 basename backfill when SQLite lacks `regexp_replace`).

**AGH gap**: Ad-hoc CREATE TABLE, no upgrade path, no embedded schema.

### 3. Dynamic UPDATE Helper with SQL Injection Prevention

```go
func BuildMapUpdate(d Dialect, table string, id uuid.UUID, updates map[string]any) (string, []any, error) {
    // Validate column names with regex (prevent SQL injection)
    // Build: UPDATE table SET col1=?, col2=? WHERE id=?
    // Auto-update: updated_at field
}
```

Dialect interface abstracts `?` (SQLite) vs `$1` (PG) placeholders.

### 4. Nullable/JSON Helpers (QUICK WIN)

```go
NilStr(s string) *string       // nil if empty
NilInt(v int) *int             // nil if zero
DerefStr(s *string) string     // "" if nil
JsonOrEmpty(data []byte) []byte        // "{}" if nil
JsonOrEmptyArray(data []byte) []byte   // "[]" if nil
```

### 5. Transaction Pattern with Defer-Rollback

```go
tx, err := db.BeginTx(ctx, nil)
if err != nil { return err }
defer tx.Rollback()  // No-op if already committed
// ... work on tx ...
return tx.Commit()
```

### 6. sqliteVal Wrapper for Complex Types

```go
func sqliteVal(v any) any {
    // maps, slices → marshal to JSON string
    // strings, ints, bools, time.Time → pass through
    b, _ := json.Marshal(v)
    return string(b)
}
```

## GoClaw vs AGH Comparison

| Aspect              | GoClaw                     | AGH                 | Gap                       |
| ------------------- | -------------------------- | ------------------- | ------------------------- |
| Connection pragmas  | Per-connection wrapper     | Query DSN params    | Potential race conditions |
| Schema versioning   | Embedded + incremental map | Ad-hoc CREATE TABLE | No upgrade path           |
| Transaction pattern | Consistent defer-rollback  | Not systematized    | Error handling varies     |
| Query building      | `base.BuildMapUpdate()`    | Inline per-store    | Code duplication          |
| Dialect abstraction | Interface + sqliteDialect  | None                | Hard to extend            |
| Nullable helpers    | Shared `base/` pkg         | Inline per file     | DRY violation             |

## Recommended Adaptations for AGH

1. **pragmaConnector** — robustness fix, ~1-2h refactor
2. **Shared base helpers** (nullable, JSON, clause builders) — ~4-6h
3. **Embedded schema + SchemaVersion tracking** — ~8-10h
4. **Formalize transaction defer-rollback** pattern across all stores
