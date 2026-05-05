---
status: pending
title: Memory Schema and Workspace DB Identity
type: backend
complexity: critical
dependencies: []
---

# Task 02: Memory Schema and Workspace DB Identity

## Overview

Lay down the durable schema and workspace identity foundation for Memory v2. This task introduces the numbered DDL changes, stable `workspace_id` semantics, and the migration path away from path-keyed workspace ownership so later storage, observability, and public surfaces have one identity model.

<critical>
- ALWAYS READ `_techspec.md` and ADR-001 through ADR-012 before implementation.
- REFERENCE the TechSpec sections `Data Models`, `Migrations`, `Config Lifecycle`, and `Greenfield Delete Targets`.
- ACTIVATE `agh-schema-migration`, `agh-code-guidelines`, and `golang-pro` before editing schema or production Go.
- MINIMIZE CODE churn outside schema registration, workspace resolution, and identity helpers.
- TESTS REQUIRED: fresh DB, migrated DB, reopen-after-restart, and idempotent backfill coverage must ship in this task.
- NO WORKAROUNDS: no `EnsureSchema` fallback, no dual `workspace_root`/`workspace_id` authority.
</critical>

<requirements>
- MUST add numbered migrations for the Slice 1 memory tables and columns: catalog extension, chunks/FTS, events, decisions, recall signals, consolidations, and any lineage columns owned by this task.
- MUST introduce stable `workspace_id` resolution via `.agh/workspace.toml` and make permission-denied resolution fail closed.
- MUST define and test the backfill path that replaces path-keyed `workspace_root` ownership with `workspace_id`.
- MUST ensure the migration plan is idempotent and safe across fresh DBs, existing DBs, and restart/replay scenarios.
- MUST keep `workspace_id` as the single durable identity for workspace-scoped memory ownership after the migration completes.
</requirements>

## Subtasks
- [ ] 2.1 Author and register all Slice 1 DDL migrations with the numbered migration framework.
- [ ] 2.2 Extend workspace resolution to own `.agh/workspace.toml` creation/loading and stable IDs.
- [ ] 2.3 Add the runtime backfill path that migrates old memory rows from `workspace_root` to `workspace_id`.
- [ ] 2.4 Remove or de-authorize the legacy path-keyed ownership model once backfill is complete.
- [ ] 2.5 Add schema and resolver tests for fresh, migrated, and restart scenarios.

## Implementation Details

See TechSpec `Data Models`, `Migrations`, `Development Sequencing` steps 1, 7, and 7b, plus `Greenfield Delete Targets`. This task should end with one durable workspace identity model and all database primitives registered, but not yet with the full new storage runtime wired.

### Relevant Files
- `internal/workspace/resolver.go` — stable workspace identity and `.agh/workspace.toml` resolution.
- `internal/workspace/workspace.go` — workspace metadata/domain helpers that may need the new ID shape.
- `internal/workspace/resolver_integration_test.go` — existing resolver tests to extend for workspace ID creation/backfill behavior.
- `internal/store/schema.go` — numbered schema registration.
- `internal/store/globaldb/migrate_workspace.go` — existing workspace migration hooks to extend or replace.
- `internal/memory/catalog.go` — current path-keyed catalog ownership that must migrate off `workspace_root`.

### Dependent Files
- `internal/store/workspacedb/*` — later task creates the per-workspace DB runtime on top of this identity model.
- `internal/memory/store.go` — later store/replay work depends on stable `workspace_id`.
- `internal/store/globaldb/global_db_observe.go` — later observability aggregation depends on the new DB topology.
- `.compozy/tasks/mem-v2/task_03.md` — storage runtime task depends on the DDL and ID model finishing here.
- `.compozy/tasks/mem-v2/task_12.md` — session lineage/ledger task depends on durable IDs and migrated schema.

### Related ADRs
- [ADR-003: Per-Workspace Catalog Database](adrs/adr-003.md) — defines the global/workspace DB split.
- [ADR-004: Stable Workspace ID via .agh/workspace.toml](adrs/adr-004.md) — defines stable workspace identity.
- [ADR-001: Hybrid Escopado as Memory Source-of-Truth Model](adrs/adr-001.md) — defines the tables that become authoritative/derived.

## Extensibility / Agent Manageability / Config Lifecycle

- Extensibility: none yet — checked surfaces are provider hooks, extension manifests, host API, and bridge SDKs; they remain unchanged until later tasks consume the new IDs.
- Agent manageability: no public route/CLI change yet, but the identity model established here constrains all later `workspaces/resolve` and memory-scope transports.
- Config lifecycle: none — checked surfaces are `config.toml`, settings routes, tool-surface writes, examples, and docs; no new keys land in this task.

### Web/Docs Impact

- `web/`: none — checked surfaces are generated types, settings pages, and knowledge/session views; they change only after public contracts stabilize.
- `packages/site`: none — checked surfaces are workspace/memory docs and API/CLI references; they update after transport semantics land.

## Deliverables

- Numbered migrations registered for all Slice 1 memory schema changes.
- Stable `.agh/workspace.toml`-backed `workspace_id` resolution.
- Backfill logic that migrates path-keyed rows to `workspace_id` and removes dual authority.
- Fresh DB and migrated DB test coverage for schema and resolver behavior.

## Tests

- Unit tests:
  - [ ] Resolver creates or loads a stable `workspace_id` from `.agh/workspace.toml`.
  - [ ] Invalid or permission-denied workspace identity resolution fails closed with deterministic errors.
  - [ ] Migration registration includes every Slice 1 memory table and column expected by the TechSpec.
- Integration tests:
  - [ ] Fresh database bootstrap reaches head with the new migrations and reopens cleanly after restart.
  - [ ] Existing path-keyed rows backfill to `workspace_id` idempotently and do not keep dual authority.
  - [ ] Re-running migrations after backfill is a no-op that preserves data and indexes.
- Test coverage target: >=80%.
- All tests must pass.

## References

- `.resources/goclaw/migrations/000001_init_schema.up.sql`
- `.resources/goclaw/migrations/000013_knowledge_graph.up.sql`
- `.resources/claude-code/tools/AgentTool/agentMemory.ts`
- `.resources/codex/codex-rs/memories/write/src/workspace.rs`

## Success Criteria

- All tests passing.
- Test coverage >=80%.
- `workspace_id` is the single durable identity for workspace-scoped memory ownership.
- All Slice 1 memory DDL is registered through numbered migrations with no schema fallback path.

