---
status: completed
title: Data layer propagation
type: backend
complexity: medium
dependencies:
  - task_01
---

# Task 03: Data layer propagation

## Overview

Propagate `StopReason` and `StopDetail` through the entire data stack: global DB schema, query functions, API contract types, conversions, and observer. After this task, stop reasons are stored in SQLite, queryable via API, and visible in session list/detail responses.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST add `stop_reason TEXT` and `stop_detail TEXT` columns to the `sessions` table
- MUST update `RegisterSession()` to include stop_reason/stop_detail in UPSERT
- MUST update `UpdateSessionState()` to conditionally include stop_reason/stop_detail in UPDATE
- MUST update `ReconcileSessions()` to handle new columns in upsert
- MUST update `scanSessionInfo()` to scan the 2 new columns (11 columns total)
- MUST add StopReason/StopDetail to `store.SessionInfo` (global DB row type, distinct from `session.SessionInfo`)
- MUST update `sessionInfoFromMeta()` in `query.go` to map StopReason from meta
- MUST add `StopReason` and `StopDetail` fields to `contract.SessionPayload`
- MUST update `SessionPayloadFromInfo()` in `conversions.go` to include stop reason fields
- MUST update `Observer.OnSessionStopped()` to pass StopReason in `SessionStateUpdate`
- MUST update `SessionStateUpdate` to include StopReason/StopDetail fields
- MUST clarify that `contract.SessionPayload.StopReason` is session-level (distinct from existing `AgentEventPayload.StopReason` which is ACP event-level)
</requirements>

## Subtasks
- [x] 3.1 Add columns to sessions table schema and write migration SQL
- [x] 3.2 Update `store.SessionInfo` with StopReason/StopDetail, update `SessionStateUpdate`
- [x] 3.3 Update `RegisterSession`, `UpdateSessionState`, `ReconcileSessions`, `scanSessionInfo`
- [x] 3.4 Update `sessionInfoFromMeta()` in `query.go` to map stop reason fields
- [x] 3.5 Add fields to `contract.SessionPayload`, update `SessionPayloadFromInfo()`
- [x] 3.6 Update `Observer.OnSessionStopped()` to pass stop reason in state update
- [x] 3.7 Write unit tests for all DB operations, conversions, and observer updates

## Implementation Details

See TechSpec "Data Models" section for field definitions and "API Endpoints" section for response format.

Note: `contract.go` already has a `StopReason` field on `AgentEventPayload` (line 95) — this is the ACP-level stop reason from agent events, NOT the session-level one. The new `StopReason` on `SessionPayload` is a different field representing why the session stopped.

### Relevant Files
- `internal/store/types.go` — `SessionInfo` struct (line 82), `SessionStateUpdate` (line 124)
- `internal/store/globaldb/global_db_session.go` — `RegisterSession` (line 12), `UpdateSessionState` (line 35), `ReconcileSessions` (line 125), `scanSessionInfo` (line 252)
- `internal/session/query.go` — `sessionInfoFromMeta()` (line 212)
- `internal/api/contract/contract.go` — `SessionPayload` (line 25)
- `internal/api/core/conversions.go` — `SessionPayloadFromInfo()` (line 18)
- `internal/observe/observer.go` — `OnSessionStopped()` (line 233)

### Dependent Files
- `internal/store/globaldb/global_db_session_test.go` — test updates for new columns
- `internal/observe/observer_test.go` — test updates for stop reason propagation
- HTTP/UDS handlers that return session data — will automatically include new fields via contract types

### Related ADRs
- [ADR-001: Canonical StopReason Enum on SessionMeta](adrs/adr-001.md) — Type lives in `internal/store`

## Deliverables
- Migration SQL adding `stop_reason` and `stop_detail` columns
- Updated global DB functions handling new columns
- Updated query, contract, conversion, and observer code
- Unit tests with 80%+ coverage **(REQUIRED)**

## Tests
- Unit tests:
  - [x] `RegisterSession` with StopReason=nil stores NULL
  - [x] `RegisterSession` with valid StopReason stores the value
  - [x] `UpdateSessionState` with StopReason updates the column
  - [x] `UpdateSessionState` without StopReason leaves column unchanged
  - [x] `scanSessionInfo` correctly reads 11 columns including stop_reason/stop_detail
  - [x] `scanSessionInfo` handles NULL stop_reason gracefully
  - [x] `ReconcileSessions` upserts sessions with stop_reason
  - [x] `sessionInfoFromMeta()` maps StopReason and StopDetail from meta
  - [x] `sessionInfoFromMeta()` handles nil StopReason (legacy meta)
  - [x] `SessionPayloadFromInfo()` includes stop_reason and stop_detail in output
  - [x] `SessionPayloadFromInfo()` omits stop_reason when empty
  - [x] Observer.OnSessionStopped passes StopReason in SessionStateUpdate
- Integration tests:
  - [x] Create session → stop → query global DB → verify stop_reason column value
  - [x] GET /api/sessions/:id returns stop_reason in JSON response
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- `make verify` passes
- Stop reasons visible in API responses for stopped sessions
- Global DB stores and queries stop reasons correctly
