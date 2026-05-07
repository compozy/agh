# TC-INT-001: Global Migration v23 - Fresh DB + Reopen-After-Restart

**Priority:** P0
**Type:** Integration
**Systems:** `internal/store/globaldb` schema, `internal/modelcatalog.Store`.
**Requirement:** TechSpec Data Model, SI-10, Task 02.
**Status:** Not Run

## Objective

Verify the migration registry creates `model_catalog_sources`, `model_catalog_rows`, `model_catalog_reasoning_efforts`, and the documented indexes on a fresh DB; that the `BEGIN IMMEDIATE` write transaction is honored; that reopening the DB after a daemon restart keeps the row identity stable; and that the migration registry append-only contract still passes after v23.

## Preconditions

- [ ] Test isolated `globaldb` instance.
- [ ] No prior migrations.

## Test Steps

1. **Fresh DB migration.**
   - Run migrator end-to-end.
   - **Expected:** `schema_migrations` ends at v23 with the documented `name`/`checksum` for the model catalog migration; previous v1-v22 unchanged.
2. **Tables and indexes exist.**
   - Inspect SQLite schema.
   - **Expected:** Three tables and the indexes `idx_model_catalog_rows_provider_model`, `idx_model_catalog_rows_source_provider`, `idx_model_catalog_sources_provider` exist; foreign-key cascade on reasoning efforts present.
3. **Insert + read round-trip.**
   - Use `Store.ReplaceSourceRows` with one row including reasoning efforts and a stale flag.
   - **Expected:** `ListRows`/`ListSourceStatus` returns identical data; reasoning efforts ordered by `rank`.
4. **Reopen after restart.**
   - Close DB; reopen.
   - **Expected:** Rows present; reasoning efforts still ordered; status row preserved.
5. **WAL/SHM companion handling.**
   - Simulate stale `-wal`/`-shm` companions; reopen.
   - **Expected:** Migrator recovers cleanly; no migration mismatch.
6. **Append-only contract guarded.**
   - Modify migration v23 hash and reopen.
   - **Expected:** Migrator fails fast with mismatch error; never silently rewrites history.

## Audit Coverage

- C6 task tree (Task 02), C8 cross-surface persistence truth.
- SI-8, SI-10.

## Pass Criteria

- All steps pass with deterministic data.

## Failure Criteria

- Schema differs from TechSpec.
- Append-only contract rewritable.
- Reopen loses rows.
