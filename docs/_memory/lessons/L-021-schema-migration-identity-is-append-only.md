# L-021 — Schema migration identity is append-only

**Class:** Persistence
**Date discovered:** 2026-05-06 (daemon restart migration integrity failure)
**Evidence sources:** Local daemon restart failure, observed `~/.agh/agh.db` `schema_migrations`
rows, `0b371eaa feat: add network threads (#105)`, `08eedb32 feat: orchestration
improvements (#106)`, and L-008 schema migration discipline

## Context

Restarting the local daemon failed before readiness with:

```text
store: migration 17 integrity mismatch: recorded "add_task_orchestration_profile_schema"/2026-05-05-add-task-orchestration-profile-schema, current "rebuild_network_conversation_containers"/2026-05-05-rebuild-network-conversation-containers
```

The live global database had already recorded:

```text
17 add_task_orchestration_profile_schema
18 add_task_review_gate_schema
19 add_notification_cursors
20 add_bridge_task_subscriptions
```

Current code had inserted `rebuild_network_conversation_containers` at version 17 and shifted the
existing task/bridge migrations to later numbers. The migration runner correctly refused to boot:
the persisted version/name/checksum identity no longer matched the binary.

## Root cause

Migration numbers were treated as a local ordering convenience instead of persisted contract data.
Fresh database tests still passed because the final schema could be built from the new order, but
an existing database carries the historical identity in `schema_migrations`. Once any developer,
QA, or release database can record a migration version/name/checksum, that identity is immutable.
Reordering the registry after that point breaks upgrades even when the end-state schema is valid.

## Rule

> SQLite migration identity is append-only. After a migration may have been applied anywhere
> meaningful, do not insert before it, reorder it, rename it, renumber it, or change its checksum.
> New schema work appends the next migration number at the registry tail.

If an existing database reports an integrity mismatch, treat it as a safety signal. Do not weaken
the runner, do not accept arbitrary mismatches, and do not manually edit `schema_migrations`.
Investigate which identity is historically valid, restore the append-only sequence, and add
observed-history upgrade coverage.

## Operationalization

- Before choosing a migration number, inspect the current registry, recent commits touching the
  registry, and relevant ledgers/tasks for concurrently landed migrations.
- New schema work appends after the highest registered version. Chronological neatness is not a
  reason to insert into the middle.
- Migration tests must include fresh database coverage and upgrade/reopen coverage. For drift
  fixes, add an observed-history regression seeded with the real `schema_migrations` prefix that
  failed in the operator database.
- Keep integrity mismatch failures strict. A mismatch means the binary and database disagree about
  history; fixing that disagreement belongs in the registry or in an ADR-backed one-pass repair.
- One-pass repair is allowed only under the existing greenfield exception: bounded to one boot,
  documented in an ADR, and followed immediately by strict semantics.

## Anti-pattern

- Inserting a new migration at an older number because it "belongs" earlier in feature chronology.
- Renumbering already-recorded migrations to make a branch merge look sequential.
- Updating tests to the new fresh-DB order without seeding an old DB and reopening it.
- Handling an integrity mismatch by allowing multiple names/checksums for one version.
- Manually updating rows in a live `schema_migrations` table to match the current binary.

## Source

- Observed local database:
  `sqlite3 /Users/pedronauck/.agh/agh.db 'SELECT version, name, checksum FROM schema_migrations ORDER BY version;'`
- Failing daemon startup:
  `error: daemon: open global database "/Users/pedronauck/.agh/agh.db": store: initialize sqlite database "/Users/pedronauck/.agh/agh.db": store: migration 17 integrity mismatch`
- `internal/store/globaldb/global_db.go` — `globalSchemaMigrations` registry
- `internal/store/schema.go` — strict `RunMigrations` version/name/checksum validation
- `docs/_memory/lessons/L-008-schema-migrations-mandatory.md`
- `0b371eaa feat: add network threads (#105)`
- `08eedb32 feat: orchestration improvements (#106)`
