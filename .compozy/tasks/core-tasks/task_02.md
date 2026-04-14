---
status: pending
title: "Persist core task and run records in `globaldb`"
type: backend
complexity: high
dependencies:
  - task_01
---

# Task 02: Persist core task and run records in `globaldb`

## Overview
Add durable storage for `Task` and `TaskRun` so the new domain can persist coordination and execution records independently from sessions. This task should establish the foundational schema, CRUD paths, and query support that later manager, API, and integration tasks can rely on.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
1. `globaldb` MUST persist `Task` and `TaskRun` records with all fields required by the TechSpec, including scope, ownership, origin, parent linkage, run status, and session attachment columns.
2. The schema MUST add indexes and query paths needed for list and lookup operations by id, scope, workspace, status, parent, owner, and channel.
3. The persistence layer MUST enforce the domain limits and nullable-field rules defined in `internal/task` rather than introducing a parallel storage-specific interpretation.
</requirements>

## Subtasks
- [ ] 2.1 Add migrations and schema definitions for task and task-run tables.
- [ ] 2.2 Implement create, get, update, and list operations for tasks in `globaldb`.
- [ ] 2.3 Implement enqueue, get, list, and state-write operations for task runs in `globaldb`.
- [ ] 2.4 Add indexes and filtering paths for scope, workspace, parent, status, owner, and channel queries.
- [ ] 2.5 Align storage validation and marshalling with the domain types introduced in `internal/task`.

## Implementation Details
Use the TechSpec "Data Models" and "API Surface" sections for the canonical field set. Follow the patterns already used by `global_db_session.go`, `global_db_automation.go`, and `global_db_network_audit.go` for schema evolution, row mapping, and integration tests.

### Relevant Files
- `internal/store/globaldb/global_db.go` — Existing store entrypoint and shared DB helpers.
- `internal/store/globaldb/global_db_session.go` — Reference for persisted runtime records and row mapping patterns.
- `internal/store/globaldb/global_db_automation.go` — Reference for scoped list/query behavior and integration test style.
- `internal/store/globaldb/migrate_workspace.go` — Reference for migration organization and boot-time schema handling.
- `internal/task/` — Source of the interfaces and validation rules this storage implementation must satisfy.

### Dependent Files
- `internal/store/globaldb/global_db_test.go` — Will likely need shared setup updates for the new schema.
- `internal/task/manager` implementation files — Will depend on the CRUD paths created here.
- `internal/daemon/boot.go` — Will rely on the new tables being migrated before service startup.

### Related ADRs
- [ADR-001: Separate Task Coordination Records from TaskRun Execution Records](../adrs/adr-001.md) — Drives the need for separate task and run persistence.
- [ADR-002: Support Global and Workspace Task Scope with Explicit Hierarchy and Bounded Dependencies](../adrs/adr-002.md) — Drives scope, parent, and channel-aware schema fields.
- [ADR-004: Support Optional Task-to-Network-Channel Binding](../adrs/adr-004.md) — Drives the nullable channel columns and filters.
- [ADR-005: Derive Actor Identity Server-Side and Allow Optional Mutable Ownership](../adrs/adr-005.md) — Drives created-by, origin, and owner field semantics.

## Deliverables
- `globaldb` schema and migrations for tasks and task runs.
- CRUD and list/query implementations for tasks and runs.
- Query filters matching the TechSpec fields and list surfaces.
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration tests for SQLite persistence behavior **(REQUIRED)**

## Tests
- Unit tests:
  - [ ] Verify row encoding and decoding preserve nullable `workspace_id`, `parent_task_id`, `owner`, and `network_channel` fields.
  - [ ] Verify create and update paths reject invalid scope combinations passed from callers.
  - [ ] Verify list queries filter correctly by scope, workspace, status, parent, owner, and channel.
- Integration tests:
  - [ ] Verify migrated databases can create and query a `global` task and a `workspace` task in the same store.
  - [ ] Verify task runs persist queued records without `session_id` and later persist attached `session_id` values correctly.
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- `globaldb` can durably store and query core task and task-run records
- Downstream manager work can rely on stable storage primitives for tasks and runs
