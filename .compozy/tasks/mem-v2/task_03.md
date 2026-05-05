---
status: pending
title: Atomic Store, Workspacedb, and Replay Core
type: backend
complexity: critical
dependencies:
  - task_01
  - task_02
---

# Task 03: Atomic Store, Workspacedb, and Replay Core

## Overview

Build the storage runtime that Memory v2 needs on top of the new contract and schema foundation. This task introduces the per-workspace DB opener, atomic file-write primitives, replay-on-boot behavior, and the core store plumbing that later controller and recall logic will use.

<critical>
- ALWAYS READ `_techspec.md` and ADR-001 through ADR-012 before implementation.
- REFERENCE the TechSpec sections `System Architecture`, `Data Models`, and `Development Sequencing` steps 4-8.
- ACTIVATE `agh-code-guidelines`, `agh-cleanup-failure-paths`, and `golang-pro` before editing production Go.
- MINIMIZE CODE churn outside storage/runtime seams; do not wire public transports in this task.
- TESTS REQUIRED: atomic-write, crash-recovery, replay, workspace DB open/migrate, and restart safety coverage must ship here.
- NO WORKAROUNDS: replay and reindex paths must use the new topology instead of special-case legacy fallbacks.
</critical>

<requirements>
- MUST add the per-workspace DB runtime (`internal/store/workspacedb`) on top of the new migration registry and workspace identity model.
- MUST introduce atomic file-write helpers and jittered `BEGIN IMMEDIATE` write helpers for memory writes.
- MUST extend the memory store to support workspace/global/agent-scoped roots on the new DB topology.
- MUST implement replay-on-boot and reindex hooks compatible with the new topology and delete targets.
- MUST keep controller-free storage logic below the controller layer so later mutation orchestration remains exclusive.
</requirements>

## Subtasks
- [ ] 3.1 Add per-workspace DB open/migrate helpers and lifecycle tests.
- [ ] 3.2 Introduce atomic file-write and jittered SQLite write helpers.
- [ ] 3.3 Extend the memory store for workspace/global/agent roots under the new topology.
- [ ] 3.4 Implement replay/reindex helpers that operate on the new DB and file layout.
- [ ] 3.5 Add crash/restart/replay coverage for the new storage runtime.

## Implementation Details

See TechSpec `Filesystem layout`, `Data Models`, and `Development Sequencing` steps 4-8. This task is the storage substrate only; the controller task later owns mutation semantics and public write orchestration.

### Relevant Files
- `internal/memory/store.go` — current memory store that must grow new topology and replay support.
- `internal/memory/catalog.go` — derived catalog/index maintenance logic that must align with the new DB runtime.
- `internal/store/sqlite.go` — shared SQLite helpers and transaction wiring.
- `internal/fileutil/*` — atomic file-write helpers used by memory file persistence.
- `internal/store/globaldb/global_db.go` — global DB conventions to mirror where appropriate.
- `internal/daemon/boot.go` — replay/reindex call sites that later tasks will wire through the daemon.

### Dependent Files
- `internal/memory/controller/*` — later controller work depends on this storage/runtime substrate.
- `internal/memory/recall/*` — recall work will query the new catalog and chunks/FTS tables.
- `internal/store/globaldb/global_db_observe.go` — later observability aggregation depends on the new workspace DB shape.
- `.compozy/tasks/mem-v2/task_05.md` — controller/WAL depends on this task finishing.
- `.compozy/tasks/mem-v2/task_12.md` — ledger work depends on durable replay-friendly storage behavior.

### Related ADRs
- [ADR-001: Hybrid Escopado as Memory Source-of-Truth Model](adrs/adr-001.md) — defines derived catalog vs durable authorities.
- [ADR-003: Per-Workspace Catalog Database](adrs/adr-003.md) — defines the global/workspace DB split.
- [ADR-004: Stable Workspace ID via .agh/workspace.toml](adrs/adr-004.md) — constrains how workspace DBs are opened and keyed.

## Extensibility / Agent Manageability / Config Lifecycle

- Extensibility: none yet — checked surfaces are provider hooks, extension host routes, and SDK contracts; they stay unchanged until later tasks expose the storage runtime.
- Agent manageability: none yet — public APIs, CLI verbs, and native tools remain unchanged in this task.
- Config lifecycle: none — checked surfaces are memory config keys, settings payloads, CLI config paths, and docs; no new public config lands here.

### Web/Docs Impact

- `web/`: none — checked surfaces are generated types and memory/session/settings UI; no public contract changes land in this task.
- `packages/site`: none — checked surfaces are runtime memory/config/workspace docs and generated references; documentation updates are deferred.

## Deliverables

- `internal/store/workspacedb` (or equivalent runtime helpers) created and migration-aware.
- Atomic file-write and jittered SQLite write helpers available for Memory v2.
- Extended memory store that supports the new DB topology, replay, and reindex behavior.
- Focused restart/replay/crash-safety tests for the storage runtime.

## Tests

- Unit tests:
  - [ ] Atomic file writes fsync/rename safely and leave no partial target files after simulated failures.
  - [ ] Workspace DB open helpers resolve the correct path and run migrations once.
  - [ ] Replay/reindex helpers reconstruct derived state deterministically from authoritative inputs.
- Integration tests:
  - [ ] Restarting after interrupted writes reopens cleanly and replays pending derived-state work.
  - [ ] Workspace/global DBs coexist without mixing rows across workspaces.
  - [ ] `go test` coverage proves no direct controller/public transport dependency leaks into the storage layer.
- Test coverage target: >=80%.
- All tests must pass.

## References

- `.resources/hermes/tools/memory_tool.py`
- `.resources/hermes/agent/curator_backup.py`
- `.resources/codex/codex-rs/memories/write/src/storage.rs`
- `.resources/codex/codex-rs/memories/write/src/runtime.rs`

## Success Criteria

- All tests passing.
- Test coverage >=80%.
- Memory v2 has atomic file/db storage primitives and replay-capable runtime support.
- The storage layer is ready for controller/recall work without public-surface drift.

