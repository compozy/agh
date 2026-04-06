---
status: completed
title: GlobalDB workspaces table and WorkspaceStore implementation
type: backend
complexity: high
dependencies:
  - task_01
---

# Task 02: GlobalDB workspaces table and WorkspaceStore implementation

## Overview

Extend the global SQLite schema with a `workspaces` table and implement `workspace.WorkspaceStore` on `store.GlobalDB`, including uniqueness constraints and session rows referencing `workspace_id` instead of a bare path string. This unlocks persistent registration hints and FK stability for sessions.

<critical>
- ALWAYS READ `_techspec.md` before starting
- REFERENCE TECHSPEC for schema and CRUD — do not paste full SQL here
- FOCUS ON "WHAT"
- MINIMIZE CODE
- TESTS REQUIRED
- GREENFIELD: Não sacrificar qualidade por retrocompatibilidade — preferir desenho limpo e mudanças breaking alinhadas ao TechSpec; evitar migrações, shims ou código defensivo só para estado/API antiga.
</critical>

<requirements>
- MUST add `workspaces` DDL to global schema per TechSpec "SQLite schema" (indexes included)
- MUST migrate session persistence from `workspace TEXT` to `workspace_id TEXT NOT NULL` referencing registered workspaces (greenfield: replace old column per project rules)
- MUST implement all `WorkspaceStore` methods on `GlobalDB` with correct constraint errors mapped to workspace package errors where applicable
- MUST enforce unique `root_dir` and unique `name` as specified
- MUST add table-driven unit tests for insert/update/delete/get/list and duplicate violations
- MUST update any global session registration paths that stored workspace path strings to use `workspace_id`
</requirements>

## Subtasks
- [x] 2.1 Add schema statements and open/migrate path for `~/.agh/agh.db`
- [x] 2.2 Implement CRUD: `InsertWorkspace`, `UpdateWorkspace`, `DeleteWorkspace`, `GetWorkspace`, `GetWorkspaceByPath`, `GetWorkspaceByName`, `ListWorkspaces`
- [x] 2.3 Change session index/meta persistence to store `workspace_id` consistently
- [x] 2.4 Extend `internal/store/global_db_test.go` with workspace and migration coverage
- [x] 2.5 Verify WAL and recovery tests still pass with expanded schema

## Implementation Details

See TechSpec "SQLite schema", "Session schema change", and "store/global_db_test.go" patterns. Reuse ID generation consistent with `sess_`/`turn_` style (`ws_` prefix per ADR-002).

### Relevant Files
- `internal/store/schema.go` — `globalSchemaStatements` extension point
- `internal/store/global_db.go` — `GlobalDB` method implementations
- `internal/store/global_db_test.go` — Existing global DB tests and helpers
- `internal/session/session.go` / meta persistence — Session field rename consumers

### Dependent Files
- `internal/workspace/store.go` — Interface satisfied by `GlobalDB`
- `internal/session/manager.go` — Will read/write `WorkspaceID` in task_04

### Related ADRs
- [ADR-002: Random Hex ID with Human-Friendly Name Field](adrs/adr-002.md) — ID generation and uniqueness constraints

## Deliverables
- `workspaces` table live in global DB
- `GlobalDB` implements `workspace.WorkspaceStore`
- Session storage uses `workspace_id`
- Unit tests in `global_db_test.go` covering CRUD and constraints **(REQUIRED)**

## Tests
- Unit tests:
  - [x] Insert workspace with unique `root_dir` and `name` succeeds
  - [x] Second insert with same `root_dir` or `name` fails with expected error
  - [x] `GetWorkspaceByPath` returns correct row after `EvalSymlinks`-consistent path storage
  - [x] `ListWorkspaces` returns all rows in stable order (document ordering)
  - [x] Session registration uses `workspace_id` and reads back correctly
- Integration tests:
  - [x] Not required beyond `global_db_test` real SQLite temp dir unless tagged `integration` elsewhere
- Test coverage target: >=80% for modified `store` surfaces
  - `go test ./internal/store -cover -count=1` reports `coverage: 80.5% of statements`
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80% for `internal/store` workspace additions
- `make verify` passes
