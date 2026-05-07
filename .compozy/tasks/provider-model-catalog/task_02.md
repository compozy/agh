---
status: completed
title: "Model Catalog Persistence"
type: backend
complexity: critical
dependencies: []
---

# Task 2: Model Catalog Persistence

## Overview
This task adds the durable SQLite foundation for model catalog source rows, status, and reasoning effort values. It keeps persistence separate from catalog merge logic so later service work can depend on a tested store boundary.

<critical>
- ALWAYS READ `_techspec.md` and every ADR before starting
- REFERENCE TECHSPEC for implementation details - do not duplicate here
- FOCUS ON "WHAT" - describe what needs to be accomplished, not how
- MINIMIZE CODE - show code only to illustrate current structure or problem areas
- TESTS REQUIRED - every task MUST include tests in deliverables
</critical>

<requirements>
- MUST append a new global SQLite migration at the tail of `globalSchemaMigrations` without reordering or modifying existing migration identities.
- MUST create `model_catalog_sources`, `model_catalog_rows`, and `model_catalog_reasoning_efforts` with the fields and indexes from the TechSpec.
- MUST make `model_catalog_sources` status rows provider-scoped; cross-provider sources synthesize one status row per AGH provider and no empty-provider sentinel exists.
- MUST store `default_reasoning_effort` as nullable data, matching other unknown metadata semantics.
- MUST include deterministic projection indexes that order by priority, freshness, and source identity.
- MUST implement transactional store operations for replacing source rows/status and listing rows/status.
- MUST preserve reasoning effort rows with side-table delete/insert inside the same transaction as source row replacement.
- MUST cover fresh DB, prior-prefix upgrade, reopen-after-restart, index presence, and append-only migration contract.
- MUST use `BEGIN IMMEDIATE` transaction patterns already used in global DB write paths.
</requirements>

## Subtasks
- [x] 2.1 Append the model catalog schema migration at the global DB registry tail.
- [x] 2.2 Add schema helper files for the new catalog tables and indexes.
- [x] 2.3 Add `GlobalDB` store methods for source row replacement, row listing, and source status listing.
- [x] 2.4 Ensure source replacement is atomic and deletes/reinserts reasoning efforts consistently.
- [x] 2.5 Add migration tests for fresh DB, prefix upgrade, reopen-after-restart, and append-only identity.
- [x] 2.6 Add store tests for filtering by provider/source/stale, provider-scoped cross-provider source status, nullable default reasoning effort, and status row updates.

## Implementation Details
Follow `_techspec.md` sections `Data Model`, `Data-Model Field Rationale`, and `Side-Table-vs-JSON Decisions`. Activate `agh-schema-migration`, `agh-code-guidelines`, `golang-pro`, `agh-test-conventions`, and `testing-anti-patterns`.

### Relevant Files
- `internal/store/globaldb/global_db.go` - global schema migration registry and base schema.
- `internal/store/globaldb/schema_notification_cursor.go` - schema sidecar pattern.
- `internal/store/globaldb/migrate_notification_cursor.go` - migration helper pattern.
- `internal/store/globaldb/global_db_notification_cursor_test.go` - migration fresh/reopen test pattern.
- `internal/store/globaldb/global_db_test.go` - append-only migration contract tests.
- `internal/store/globaldb/tx_helpers.go` - transaction helper patterns.
- `internal/store/schema.go` - common migration runner behavior.

### Dependent Files
- `internal/modelcatalog` - Task 03 will define or consume shared row/status types.
- `internal/store/globaldb/global_db_extra_test.go` - may need table presence assertions.
- `internal/store/types.go` - only if shared store type definitions need extension; prefer `internal/modelcatalog` types when possible.

### Related ADRs
- [ADR-001: Daemon-Owned Provider Model Catalog](adrs/adr-001-daemon-owned-provider-model-catalog.md) - requires daemon-owned persisted catalog rows and status.

### Web/Docs Impact
- `web/`: none directly in this task - checked persistence-only global DB changes; web consumes catalog through API in Task 09.
- `packages/site`: none directly in this task - schema is internal and documented through public model catalog behavior in Task 10.

### Extensibility / Agent Manageability / Config Lifecycle
- Extensibility: stores extension-provided source rows later, but no extension method is added here.
- Agent manageability: no direct CLI/HTTP/UDS surface here; persisted status enables later agent-operable status endpoints.
- Config lifecycle: no TOML changes in this task; persistence supports catalog rows derived from config in Task 03.

## Deliverables
- Append-only global DB migration for model catalog tables/indexes.
- Transactional global DB store methods for source rows/status/reasoning efforts.
- Fresh DB, upgrade, reopen-after-restart, and append-only tests **(REQUIRED)**.
- Store filtering/status tests with 80%+ coverage **(REQUIRED)**.

## Tests
- Unit tests:
  - [x] migration creates all three catalog tables and required indexes.
  - [x] source row replacement removes stale rows for the same source/provider.
  - [x] reasoning efforts are replaced atomically with their parent rows.
  - [x] row listing filters by provider ID, source ID, and stale/include-stale flags.
  - [x] source status listing returns row count, stale flag, and redacted last error.
  - [x] `models_dev` status rows are stored per provider without a `provider_id=''` sentinel.
  - [x] `default_reasoning_effort` round-trips as NULL when unknown.
  - [x] provider/model listing is deterministic for equal freshness using source identity.
- Integration tests:
  - [x] fresh global DB opens with catalog schema present.
  - [x] DB seeded through the previous migration prefix upgrades to the new catalog migration.
  - [x] reopen-after-restart does not reapply or mutate migration records.
  - [x] append-only migration identity test includes the new version at the tail.
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- `go test ./internal/store/globaldb` passes.
- Catalog schema is versioned only through the migration registry, not `EnsureSchema`-style reconciliation.
