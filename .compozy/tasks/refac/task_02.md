---
status: pending
title: File-level splits (all bloated files)
type: refactor
complexity: medium
dependencies:
  - task_01
---

# Task 02: File-level splits (all bloated files)

## Overview

Split seven oversized files (all >500 LOC) into focused, single-responsibility files within the same package. This is pure file-level reorganization — all methods stay on existing receiver types, no public API changes, no import changes for consumers. The `udsapi/handlers.go` split is especially important as it becomes a prerequisite for the `apicore/` extraction in task 03.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST split `daemon/daemon.go` (1,495 LOC) into at least 5 focused files
- MUST split `session/manager.go` (1,205 LOC) into at least 4 focused files
- MUST split `store/global_db.go` (1,097 LOC) into at least 4 files by data domain
- MUST split `workspace/resolver.go` (1,078 LOC) into at least 4 focused files
- MUST split `udsapi/handlers.go` (1,058 LOC) to match httpapi's file organization
- MUST split `store/schema.go` (868 LOC) into DDL, SQLite infrastructure, and migration files
- MUST split `store/store.go` (568 LOC) into types and SQL helpers
- MUST NOT change any function signatures, receiver types, or package-level exports
- MUST NOT change any behavior — pure file reorganization
- All existing tests MUST pass unchanged
- `udsapi/` target file layout MUST mirror `httpapi/` naming conventions
</requirements>

## Subtasks

- [ ] 2.1 Split `daemon/daemon.go` → `daemon.go`, `boot.go`, `dream.go`, `orphan.go`, `boundary.go`, `notifier.go`
- [ ] 2.2 Split `session/manager.go` → `manager.go`, `manager_lifecycle.go`, `manager_prompt.go`, `manager_workspace.go`, `manager_helpers.go`
- [ ] 2.3 Split `store/global_db.go` → `global_db.go`, `global_db_workspace.go`, `global_db_session.go`, `global_db_observe.go`, `global_db_permission.go`
- [ ] 2.4 Split `store/schema.go` → `schema.go`, `sqlite.go`, `migrate_workspace.go`
- [ ] 2.5 Split `store/store.go` → `types.go`, `store.go`, `sql_helpers.go`
- [ ] 2.6 Split `workspace/resolver.go` → `resolver.go`, `resolver_crud.go`, `scanner.go`, `clone.go`, `helpers.go`
- [ ] 2.7 Split `udsapi/handlers.go` → `sessions.go`, `agents.go`, `observe.go`, `daemon.go`, `stream.go`, `prompt.go`, `payloads.go`

## Implementation Details

See TechSpec "Phase 2: File-Level Splits" items 2.1–2.7. See individual refactoring reports for function-to-file mapping:
- [Infra report](./20260406-config-daemon-cli.md) F3 — daemon.go (38+ functions with line ranges)
- [Core report](./20260406-core-session-acp.md) F1 — manager.go (49+ functions)
- [Storage report](./20260406-storage-observe-memory.md) F1, F4, F5 — store files (34+29+32 functions)
- [New report](./20260406-skills-workspace.md) F2 — resolver.go (42+ functions)
- [API report](./20260406-api-layer.md) F2 — handlers.go (target layout from httpapi)

### Relevant Files

- `internal/daemon/daemon.go` (1,495 lines) — boot, dream loop, orphan cleanup, boundary verification, process utils, notifier
- `internal/session/manager.go` (1,205 lines) — lifecycle, workspace resolution, prompting, recording, cleanup
- `internal/store/global_db.go` (1,097 lines) — workspace CRUD, session registry, observability, permissions
- `internal/workspace/resolver.go` (1,078 lines) — CRUD, resolution, caching, scanning, cloning, ID gen
- `internal/udsapi/handlers.go` (1,058 lines) — all handlers, payloads, SSE, parsers, conversions
- `internal/store/schema.go` (868 lines) — DDL, SQLite infra, legacy migration
- `internal/store/store.go` (568 lines) — domain types, Validate methods, SQL helpers

### Dependent Files

- No dependent files — file-level splits within the same package have zero consumer impact
- `udsapi/` target layout mirrors `httpapi/`: `sessions.go`, `agents.go`, `observe.go`, `prompt.go`, `stream.go`, `daemon.go`

## Deliverables

- `daemon/` package: 6+ files instead of 1 monolith
- `session/` package: 5+ files instead of 1 monolith
- `store/` package: 7+ files instead of 3 monoliths
- `workspace/` package: 5+ files instead of 1 monolith
- `udsapi/` package: 7+ handler files matching httpapi layout
- All existing tests pass unchanged **(REQUIRED)**
- `make verify` passes **(REQUIRED)**

## Tests

- Unit tests:
  - [ ] All existing `daemon/` tests pass without modification
  - [ ] All existing `session/` tests pass without modification
  - [ ] All existing `store/` tests pass without modification
  - [ ] All existing `workspace/` tests pass without modification
  - [ ] All existing `udsapi/` tests pass without modification
  - [ ] `make test -race` passes for all packages
- Integration tests:
  - [ ] `make test` full suite passes
- Test coverage target: >=80% (maintained from existing coverage)

## Success Criteria

- All tests passing
- Test coverage >=80% (unchanged)
- `make verify` passes
- No file exceeds 400 lines in split packages
- `udsapi/` file layout matches `httpapi/` naming conventions
- Zero changes to any function signature or public API
