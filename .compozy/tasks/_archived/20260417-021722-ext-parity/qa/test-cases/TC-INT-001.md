# TC-INT-001: Fresh database creates resource tables deterministically

**Priority:** P0
**Type:** Integration
**Package:** internal/store/globaldb
**Related Tasks:** 01

## Objective

Validate that booting globaldb on an empty SQLite database deterministically creates the `resource_records` and `resource_source_state` tables with all expected columns and indexes. This is the foundational schema gate — every downstream feature depends on these tables existing with the correct shape after a clean first boot.

## Preconditions

- Empty directory via `t.TempDir()` for SQLite database path
- No pre-existing database files
- globaldb package imported and available

## Test Steps

1. Open globaldb against the empty `t.TempDir()` path.
   **Expected:** No error. Database file created on disk.

2. Query `sqlite_master` for table names matching `resource_records` and `resource_source_state`.
   **Expected:** Both tables exist. No other `resource_*` tables are present unless explicitly expected.

3. Query `pragma table_info('resource_records')` to inspect column definitions.
   **Expected:** Columns include at minimum: `kind`, `id`, `source`, `owner_kind`, `owner_id`, `data`, `created_at`, `updated_at`. Types and nullability match the schema definition.

4. Query `pragma table_info('resource_source_state')` to inspect column definitions.
   **Expected:** Columns include at minimum: `source`, `kind`, `nonce`, `updated_at`. Types and nullability match the schema definition.

5. Query `pragma index_list('resource_records')` and `pragma index_list('resource_source_state')` to enumerate indexes.
   **Expected:** Indexes exist for common query patterns — at minimum a unique index on `(kind, id)` for resource_records and on `(source, kind)` for resource_source_state.

6. Close the database and reopen it against the same path.
   **Expected:** No migration errors. Schema is identical to step 2-5 (idempotent boot).

## Edge Cases

- Boot twice in rapid succession on the same path — no "table already exists" errors
- Boot with a corrupted or zero-byte database file — returns clear error, does not panic
- Verify WAL mode is enabled if the project requires it for concurrent reads
- Column ordering in `pragma table_info` should be deterministic across runs
