# Migration Decision Matrix

Use the matrix below to decide what kind of artifact your change needs.

| Change | Migration required? | Notes |
|--------|---------------------|-------|
| Add a NOT NULL column to existing table | YES | Provide a default expressible in SQL or backfill in the same transaction. |
| Add a NULLable column | YES | Even nullable adds need to be migrated, not implicit. |
| Drop a column | YES | SQLite supports `ALTER TABLE DROP COLUMN` since 3.35; check the project's minimum version before using. |
| Rename a column | YES | Use `CREATE NEW + INSERT INTO ... SELECT + DROP + RENAME` if direct rename isn't supported. |
| Add an index | YES | Idempotent: `CREATE INDEX IF NOT EXISTS`. |
| Drop an index | YES | `DROP INDEX IF EXISTS`. |
| Add a CHECK constraint | YES | Same as table-rebuild for older SQLite versions. |
| Add a unique constraint | YES | Risk of breaking existing data — surface in techspec. |
| New table | YES | Idempotent: `CREATE TABLE IF NOT EXISTS`. |
| Drop a table | YES | Greenfield-alpha: hard cut. Mention the delete target in the techspec. |
| Add a row (seed data) | YES | Use `INSERT OR IGNORE` for idempotence. |
| Change default value | YES | Existing rows are unaffected by SQLite default changes — explicit backfill if needed. |
| Touch struct field that round-trips through SQLite | YES (column add/rename) | The Go struct change is just the front of the migration. |
| In-memory cache shape change | NO | This skill does not apply. |
| `internal/memory/MEMORY.md` schema | NO | Markdown is the source of truth; FTS5 catalog is derived (see `docs/_memory/analysis/analysis_codex_plans.md`). Reindex via `internal/memory/consolidation`. |

## Greenfield rule

If the migration would require a "preserve old behavior" branch, the answer is "delete the old thing." Hard-cut renames sweep code, storage, APIs, CLI, extensions, specs, RFCs, AND `.compozy/tasks/*` artifacts in the same change.

The narrow exception: in-place ALTER + one-shot repair when the cost of "delete the old thing" is "every developer rebuilds their local SQLite." Repair is bounded to a single boot, strict semantics resume immediately, exception is documented in an ADR (see `session-driver-override/adrs/adr-005.md`).
