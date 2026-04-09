---
status: pending
title: StopReason + StopCause types
type: backend
complexity: medium
dependencies: []
---

# Task 01: StopReason + StopCause types

## Overview

Define the foundational types for session resilience: a `StopReason` enum in `internal/store` (co-located with `SessionMeta` to avoid import cycles) and a `StopCause` enum in `internal/session` that explicitly signals why a stop was requested. Extend `SessionMeta`, `Session`, and `SessionInfo` with stop reason fields. This task creates the data model that all subsequent tasks build on.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST define `StopReason` as a string type with 10 constants in `internal/store/types.go`
- MUST define `ValidStopReason()` function that returns true for valid enum members
- MUST add `StopReason *StopReason` and `StopDetail string` fields to `SessionMeta` with JSON tags
- MUST update `SessionMeta.Validate()` to reject invalid StopReason values (when non-nil)
- MUST define `StopCause` as an int type with 6 constants in `internal/session/stop_cause.go`
- MUST add `stopCause`, `stopReason`, `stopDetail` fields to `Session` struct
- MUST add `StopReason` and `StopDetail` fields to `session.SessionInfo`
- MUST update `Session.Info()` to include stop reason fields in the snapshot
- MUST update `Session.Meta()` to include stop reason fields in the meta output
- MUST update `ReadSessionMeta()` and `WriteSessionMeta()` to handle new fields
</requirements>

## Subtasks
- [ ] 1.1 Define `StopReason` type, 10 constants, and `ValidStopReason()` in `internal/store/types.go`
- [ ] 1.2 Add `StopReason`/`StopDetail` fields to `SessionMeta`, update `Validate()`
- [ ] 1.3 Define `StopCause` type and 6 constants in new file `internal/session/stop_cause.go`
- [ ] 1.4 Add stop fields to `Session` struct and update `Info()` and `Meta()` methods
- [ ] 1.5 Verify `ReadSessionMeta`/`WriteSessionMeta` round-trip with new fields
- [ ] 1.6 Write unit tests for all new types and validation

## Implementation Details

See TechSpec "Core Interfaces" and "Data Models" sections for exact type definitions and constant values.

### Relevant Files
- `internal/store/types.go` — `SessionMeta` struct (line 287), `SessionInfo` struct (line 82), `Validate()` methods
- `internal/store/meta.go` — `ReadSessionMeta()`, `WriteSessionMeta()` for JSON persistence
- `internal/session/session.go` — `Session` struct (line 59), `SessionInfo` (line 45), `Info()` (line 86), `Meta()` (line 354)

### Dependent Files
- `internal/store/globaldb/global_db_session.go` — will need schema updates (task 03)
- `internal/session/manager_lifecycle.go` — will use StopCause for classification (task 02)
- `internal/session/query.go` — will map StopReason from meta (task 03)

### Related ADRs
- [ADR-001: Canonical StopReason Enum on SessionMeta](adrs/adr-001.md) — Type ownership in `internal/store`, explicit StopCause mechanism

## Deliverables
- `StopReason` type with 10 constants and `ValidStopReason()` in `internal/store/types.go`
- `StopCause` type with 6 constants in `internal/session/stop_cause.go`
- Extended `SessionMeta`, `Session`, `SessionInfo` with stop reason fields
- Updated `Info()`, `Meta()`, `Validate()` methods
- Unit tests with 80%+ coverage **(REQUIRED)**

## Tests
- Unit tests:
  - [ ] All 10 StopReason constants pass `ValidStopReason()`
  - [ ] Empty string and arbitrary strings fail `ValidStopReason()`
  - [ ] `SessionMeta.Validate()` passes when StopReason is nil
  - [ ] `SessionMeta.Validate()` passes when StopReason is valid
  - [ ] `SessionMeta.Validate()` fails when StopReason is invalid string
  - [ ] `Session.Info()` includes StopReason and StopDetail in snapshot
  - [ ] `Session.Meta()` includes StopReason and StopDetail in output
  - [ ] `ReadSessionMeta`/`WriteSessionMeta` round-trip preserves StopReason and StopDetail
  - [ ] `ReadSessionMeta` of legacy meta without StopReason fields succeeds (nil StopReason)
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- `make verify` passes
- `StopReason` type usable from both `internal/store` and `internal/session` without import cycles
