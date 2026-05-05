---
status: completed
title: Observability Retention and Health Base
type: backend
complexity: high
dependencies:
  - task_01
---

# Task 02: Observability Retention and Health Base

## Overview

Add the retention and health payload foundation required by Hermes observability issues. This task wires `observability.retention_days` into a real sweep path, extends observe health data without overloading unrelated APIs, and prepares shared health fields later lifecycle and memory tasks can use.

<critical>
- ALWAYS READ `_techspec.md`, ADR-001, and task_01 outputs before changing observe storage
- DO NOT add unbounded background goroutines; all sweeps must be owned and cancellable
- DO NOT delete fresh events or session evidence needed by active debugging
- KEEP retention behavior deterministic in tests by injecting time
- API and SSE payload changes must stay typed through `internal/api/contract`
</critical>

<requirements>
- MUST implement retention using `observability.retention_days`
- MUST keep retention disabled or no-op when the configured value means keep history
- MUST expose health fields needed by selected Hermes issues without leaking internal-only state
- MUST add tests for sweep cutoff, no-op retention, and API conversion behavior
- MUST document any operator-visible changes in CLI, `web/`, or site docs
- MUST analyze and implement required `web/` and `packages/site` follow-up changes caused by this task
</requirements>

## Subtasks
- [x] 2.1 Trace current observe event persistence and identify the single owner for retention sweeps
- [x] 2.2 Implement retention cutoff logic with injected clock and explicit context cancellation
- [x] 2.3 Extend health read models and API conversions for retention and persistence health fields
- [x] 2.4 Add unit and integration tests for retention, health payloads, and configuration edge cases
- [x] 2.5 Update CLI/API/web/docs surfaces that expose observability health or retention behavior
- [x] 2.6 Analyze and implement any required follow-up changes in `web/` and `packages/site`, including documentation, typed clients, settings pages, examples, stories, and tests where applicable

## Implementation Details

Retention should be a store-owned operation invoked by an explicit lifecycle owner, not an implicit loop hidden in a data package. Health payloads should provide enough signal for automation, lifecycle, and memory tasks to report degraded state later while keeping contract DTOs stable and explicit.

### Relevant Files
- `internal/config/config.go` - `observability.retention_days` definition and validation
- `internal/observe/` - health and query behavior
- `internal/store/globaldb/global_db_observe.go` - observe persistence and retention sweep destination
- `internal/api/contract/contract.go` - typed health payloads
- `internal/api/core/conversions.go` - observe health conversion logic
- `internal/cli/observe.go` - CLI health or observe output if operator-visible

### Dependent Files
- `internal/observe/*_test.go` - observe health and retention tests
- `internal/store/globaldb/*observe*_test.go` - SQLite retention coverage
- `internal/api/core/*_test.go` - API conversion tests for health payloads
- `web/src/` - typed client or settings display updates if health/retention fields surface in the app
- `packages/site/` - operator docs for retention and health behavior when user-visible
- `.compozy/tasks/hermes/task_03.md` - lifecycle failure observability builds on this health base
- `.compozy/tasks/hermes/task_07.md` - memory health builds on this health base

### Related ADRs
- [ADR-001: Hermes Hardening Tracks](adrs/adr-001-hermes-hardening-tracks.md) - requires observability retention and health to precede dependent tracks

## Deliverables
- Config-driven observability retention sweep
- Extended typed health payloads for retention and persistence state
- Tests proving cutoff behavior, disabled retention behavior, and API conversions
- Updated operator-facing docs or UI when new health fields are surfaced
- Documented `web/` and `packages/site` impact assessment with required changes applied or explicitly marked not applicable

## Tests
- Unit tests:
  - [x] Retention cutoff keeps events newer than the configured window
  - [x] Disabled retention does not delete events
  - [x] Health conversion preserves typed fields and omits no required data
  - [x] Invalid retention configuration fails validation clearly
- Integration tests:
  - [x] SQLite observe store deletes only eligible rows
  - [x] API health endpoint or handler returns the new health payload shape
  - [x] Existing observe query tests continue to pass with retained data
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- Retention is real, tested, and controlled by configuration
- Health payloads are ready for lifecycle and memory tracks
- No hidden goroutine or nondeterministic sweep logic is introduced
- Relevant backend tests pass after the change
