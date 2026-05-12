---
name: agh-schema-migration
description: Authors numbered schema migrations for AGH SQLite databases (agh.db, events.db, catalog DBs) whenever a column, index, constraint, or table changes. Enforces the migration registry primitive over EnsureSchema-style boot reconciliation, requires schema_migrations recording, transactional wrap, idempotence, and tests for fresh-DB and reopen-after-restart paths. Verifies SQLite recovery code renames -wal and -shm companions. Use when editing internal/store, internal/store/globaldb, internal/store/sessiondb, internal/memory persistence, internal/automation scheduler state, or any code that issues DDL. Do not use for in-memory data structures or non-SQLite caches.
trigger: implicit
---

# AGH Schema Migration

Hermes BUG-002 (Critical) cost a full review round because `internal/memory/catalog.go` widened `memory_operation_log` via `EnsureSchema` — fresh installs worked, upgrades failed with `no such column: scope`. Greenfield-alpha policy still requires that "delete the old thing" decisions be made explicitly; one-pass repair is allowed only when documented in an ADR. Default: real numbered migration in the registry.

## Procedures

**Step 1: Classify the Change**

1. Identify what is changing: column add/remove/rename, index add/remove, constraint add/remove, new table, dropped table.
2. Confirm the change actually requires a migration. Read `references/migration-decision.md` for the decision matrix.
3. If the change is purely in-memory (struct field) and never round-trips through SQLite, this skill does not apply.

**Step 2: Locate the Migration Registry**

1. Read `internal/store/` for the canonical migrations runner and the registry of numbered migrations.
2. Identify the next migration number (sequential, gap-free).
3. Identify which database(s) the migration applies to: `agh.db` (global), `events.db` (per-session), automation scheduler state, memory operation log.
4. Confirm a SHARED migration primitive across all SQLite databases. If a database still uses `EnsureSchema`-style boot reconciliation, refactor that database to use the migration registry FIRST as a separate task.

**Step 3: Author the Migration**

1. Read `references/migration-template.md` for the canonical migration shape.
2. Write the migration as a numbered Go file (or SQL string registered programmatically) following the existing style.
3. Wrap in a transaction (`BEGIN IMMEDIATE`).
4. Make the operations idempotent where possible (`CREATE INDEX IF NOT EXISTS`, `ADD COLUMN` only after a guard query).
5. Record the applied migration in `schema_migrations`.
6. For `ADD COLUMN`, default values are explicit (`NOT NULL DEFAULT '...'`). Backfill in the same transaction when defaults can't be expressed in SQL.

**Step 4: Hard-Cut vs One-Pass Repair**

1. Default policy (greenfield-alpha): if the change breaks existing data shapes, prefer "hard cut + dev wipes local SQLite" rather than writing repair code. Document the delete target in the techspec.
2. One-pass repair is allowed ONLY when:
   - The cost of "delete the old thing" is "every developer rebuilds their local SQLite" (real friction).
   - Repair is bounded to a single boot.
   - Strict semantics resume immediately after repair.
   - The exception is documented in an ADR (reference: `session-driver-override/adrs/adr-005.md`).
3. NEVER write open-ended compat code or dual-shape branches.

**Step 5: SQLite Recovery Hygiene**

1. If the migration touches recovery paths (e.g., `recoverSQLiteDatabase`), confirm `-wal` and `-shm` companions are renamed alongside the `.db` file. The refac-v2 issue #001 was Critical because only `.db` was renamed, leaving stale WAL pages.
2. Use `BEGIN IMMEDIATE` for atomic claim/lease, not `BEGIN`.
3. Watch for `ORDER BY 0` — SQLite parses positional integers in `ORDER BY` as column references, not literals. Use `(SELECT 0)` or an explicit constant column.

**Step 6: Test the Migration**

1. Write `Test<Migration>FreshDB` — opens an empty database, applies migrations, asserts schema.
2. Write `Test<Migration>ReopenAfterRestart` — opens with the previous migration applied, runs the new migration, asserts schema and that previous data is preserved (or correctly transformed).
3. Read `references/migration-test-patterns.md` for additional cases.
4. Run `go test ./internal/store/... -count=1 -race` then `make verify`.

## Error Handling

- **`EnsureSchema`-only database:** the entire database needs to be moved to the migration registry. Refactor as a separate task; do not stack a real migration on top of an `EnsureSchema` shell.
- **Column rename:** SQLite has limited rename support. Use the canonical `CREATE NEW + INSERT INTO ... SELECT + DROP + RENAME` dance. Document in the migration comment.
- **Existing migration uses non-transactional DDL:** SQLite treats most DDL transactionally, but check that the previous migration didn't open a long-lived statement that would block. If in doubt, wrap in `BEGIN IMMEDIATE`.
- **Migration fails mid-application on user DB:** the migration must be idempotent enough that a retry from `schema_migrations` can resume. Document failure modes in the migration body.
- **Schema-version constants in Go code that don't match the registry:** a CLAUDE.md violation. The registry is the single source of truth.
