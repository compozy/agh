---
status: completed
title: Split persistence into store, store/sessiondb, and store/globaldb
type: refactor
complexity: critical
dependencies:
  - task_05
---

# Task 06: Split persistence into store, store/sessiondb, and store/globaldb

## Overview
This task establishes the explicit persistence boundary defined in the TechSpec by separating per-session and global SQLite responsibilities. It is a cross-cutting refactor because `session`, `workspace`, `observe`, and `daemon` all depend on the current `internal/store` package.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- `internal/store/sessiondb` MUST own per-session event persistence and writer-loop lifecycle.
- `internal/store/globaldb` MUST own global registry, workspace persistence, observability summaries, permission logs, and token stats.
- `internal/store` MUST shrink to shared types, validation, and narrow interfaces rather than remaining a broad persistence package.
- This task MUST use real SQLite-backed tests and MUST run both `make verify` and `make test-integration`.
</requirements>

## Subtasks
- [x] 6.1 Create `internal/store/sessiondb` and move per-session database ownership into it.
- [x] 6.2 Create `internal/store/globaldb` and move global registry and workspace-backed persistence into it.
- [x] 6.3 Narrow the shared interfaces and helper surface left in `internal/store`.
- [x] 6.4 Update runtime consumers to the new persistence package ownership.
- [x] 6.5 Remove any transitional bridges before closing the task.

## Implementation Details
Use the TechSpec `Component Overview`, `Data Models`, and `Build Order` sections. Keep concrete persistence types concrete. Do not introduce an abstract repository framework. Preserve real SQLite coverage and package-local ownership of the correct database lifecycle.

### Relevant Files
- `internal/store/session_db.go` — Owns per-session event persistence and writer-loop behavior.
- `internal/store/global_db.go` — Owns the global registry database lifecycle.
- `internal/store/global_db_session.go` — Contains global session index operations that belong with `globaldb`.
- `internal/store/global_db_workspace.go` — Contains workspace persistence operations that belong with `globaldb`.
- `internal/store/store.go` — Defines the shared interfaces and helpers that must be narrowed.

### Dependent Files
- `internal/session/manager.go` — Depends on event recording and session lookup surfaces.
- `internal/workspace/resolver.go` — Depends on workspace persistence and lookup surfaces.
- `internal/observe/observer.go` — Depends on global summary and token stats persistence.
- `internal/daemon/daemon.go` — Wires both persistence surfaces together in the composition root.
- `internal/store/session_db_integration_test.go` — Must keep real SQLite session-db behavior covered through the split.

### Related ADRs
- [ADR-001: Adopt a Broad Package-Graph Reorganization for Refac V2](../adrs/adr-001.md) — Defines broader package-boundary cleanup.
- [ADR-003: Split Persistence into store/sessiondb and store/globaldb](../adrs/adr-003.md) — Establishes the target persistence structure.
- [ADR-004: Use Phased Cutovers with Same-Phase Bridge Removal and Layered Verification](../adrs/adr-004.md) — Requires verify and integration gates for structural runtime phases.

## Deliverables
- `internal/store/sessiondb` package owning per-session persistence.
- `internal/store/globaldb` package owning global registry and workspace-backed persistence.
- `internal/store` reduced to shared types, validation, and narrow interfaces.
- Updated runtime consumers and real SQLite test coverage with `make verify` and `make test-integration` passing.

## Tests
- Unit tests:
  - [x] Session-db writes still increment sequence ordering and preserve token usage semantics after re-rooting.
  - [x] Global-db session registry operations still register, update, and list sessions correctly.
  - [x] Global-db workspace persistence still handles lookup, uniqueness, and deletion constraints correctly.
  - [x] Shared `store` interfaces and validation helpers remain narrow and correct after the split.
- Integration tests:
  - [x] Session persistence integration tests still pass against real SQLite files after the package split.
  - [x] Workspace and observe flows still pass through the global database surface after the split.
  - [x] Daemon runtime integration still boots and exercises session creation with the split persistence packages.
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- Per-session and global SQLite ownership are explicit in separate packages
- `internal/store` no longer acts as a broad catch-all persistence package
