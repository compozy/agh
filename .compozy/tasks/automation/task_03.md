---
status: completed
title: Implement dispatcher, run recording, and execution governance
type: backend
complexity: high
dependencies:
  - task_02
---

# Task 03: Implement dispatcher, run recording, and execution governance

## Overview

Implement the shared execution path that turns an approved automation activation into a session plus a persisted run record. This task centralizes global concurrency control, retry behavior, and restart-safe fire-limit decisions so every activation path behaves consistently.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
1. MUST implement a shared dispatcher that is the only automation execution path for schedules, triggers, manual job fires, and later extension-fired events.
2. MUST enforce the global concurrency limit inside the dispatcher or manager layer rather than relying on scheduler-local protections.
3. MUST record run lifecycle state transitions and retry attempts in the persistence layer so restart-safe fire-limit evaluation can reuse them.
4. MUST create sessions with the correct `session.CreateOpts` for global and workspace-scoped automations, including prompt handoff behavior expected by the TechSpec.
</requirements>

## Subtasks
- [x] 3.1 Create the dispatcher and the narrow session-creation interface it depends on
- [x] 3.2 Add run lifecycle recording for scheduled, running, completed, failed, and cancelled states
- [x] 3.3 Add global concurrency gating shared by all dispatch callers
- [x] 3.4 Add retry orchestration and restart-safe fire-limit checks backed by persisted runs
- [x] 3.5 Add tests proving one execution path is shared across activation sources

## Implementation Details

Use the TechSpec sections "Dispatcher", "Development Sequencing", "Testing Approach", and "Monitoring and Observability" as the execution contract. Keep dispatch ownership in `internal/automation/` and consume the store interfaces from task 02 instead of embedding persistence details in runtime logic.

### Relevant Files
- `internal/session/manager.go` — Defines the runtime session creation behavior and `CreateOpts` shape that dispatch must respect
- `internal/session/interfaces.go` — Existing session interfaces show the narrow subset automation should depend on
- `internal/memory/consolidation/runtime.go` — Uses the same `session.CreateOpts` flow and is a useful precedent for background session spawning
- `internal/store/globaldb/global_db.go` — The new automation persistence added in task 02 will back run recording and fire-limit queries

### Dependent Files
- `internal/automation/scheduler.go` — Scheduled jobs will invoke this dispatcher in the next task
- `internal/automation/trigger.go` — Trigger matches will also invoke the same dispatcher
- `internal/daemon/boot.go` — The composed automation manager will inject the dispatcher into the runtime later

### Related ADRs
- [ADR-002: Unified Automation Model — Schedules and Triggers](adrs/adr-002.md) — Requires one shared dispatch and execution-governance path
- [ADR-004: Configurable Per-Job Retry with Fire Limits](adrs/adr-004.md) — Constrains retry strategy and fire-limit behavior

## Deliverables
- Shared dispatcher implementation in `internal/automation/`
- Persisted run lifecycle transitions and retry-aware execution governance
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration tests for concurrency, run recording, and fire-limit behavior **(REQUIRED)**

## Tests
- Unit tests:
  - [x] Dispatching a workspace-scoped automation uses the expected workspace-aware `session.CreateOpts`
  - [x] Dispatching a global automation omits workspace binding and still records the run correctly
  - [x] Exceeding the configured global concurrency limit rejects a dispatch before session creation
  - [x] Exceeding the persisted fire-limit window rejects a dispatch even after the dispatcher is recreated
  - [x] A failed run with `backoff` retry settings records the next attempt and retry metadata correctly
- Integration tests:
  - [x] Two concurrent dispatch requests from different activation kinds share the same concurrency gate
  - [x] A successful dispatch records scheduled/running/completed state transitions that can be queried back from the store
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- Every automation execution path can route through one dispatcher
- Global concurrency and persisted fire-limit behavior are enforced in runtime rather than by caller convention
