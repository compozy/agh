---
status: pending
title: "Implement `TaskRun` lifecycle and propagated cancellation"
type: backend
complexity: critical
dependencies:
  - task_04
---

# Task 05: Implement `TaskRun` lifecycle and propagated cancellation

## Overview
Implement the execution-side lifecycle that turns stored task coordination into controlled, auditable work execution. This task establishes manager-owned run transitions, run-to-task reconciliation, and the cancellation model that must propagate through task trees without leaking live work.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
1. `TaskRun.status` MUST be manager-owned and transition only through the queue-first lifecycle accepted in the TechSpec.
2. The manager MUST reconcile canonical task status from dependency state, run state, and explicit lifecycle actions instead of trusting direct patches.
3. Cancelling a parent task MUST propagate through open subtasks, queued runs, and active runs according to the cooperative-then-forced model defined in the TechSpec.
</requirements>

## Subtasks
- [ ] 5.1 Implement enqueue, claim, start, complete, fail, and cancel operations for `TaskRun`.
- [ ] 5.2 Implement task-status reconciliation from dependencies and active or terminal run state.
- [ ] 5.3 Implement cancellation propagation across task trees, including queued work and active runs.
- [ ] 5.4 Persist lifecycle and cancellation audit events through the task store.
- [ ] 5.5 Implement manager-side state gating that prevents invalid run transitions and arbitrary status mutation.

## Implementation Details
Use the TechSpec sections "Lifecycle Model", "Run Authority and Attachment Rules", "Mutability Rules", and "Cancellation Model". This task should leave transport and session concerns out of the manager core except through the interfaces defined in `internal/task`.

### Relevant Files
- `internal/task/` — Manager lifecycle, reconciliation, and cancellation implementation files.
- `internal/store/globaldb/` — Provides persisted run, dependency, and event records consumed by lifecycle logic.
- `internal/session/stop_reason.go` — Reference existing stop/cancellation vocabulary that task cancellation should align with.
- `.compozy/tasks/core-tasks/_techspec.md` — Source of canonical run states and cancellation rules.

### Dependent Files
- `internal/daemon/boot.go` — Will later wire the session bridge into these lifecycle transitions.
- `internal/api/core/` — Will expose these lifecycle operations through handlers and transports.
- `internal/observe/` — Will consume lifecycle events and reconciled status outputs for metrics and health.

### Related ADRs
- [ADR-001: Separate Task Coordination Records from TaskRun Execution Records](../adrs/adr-001.md) — Governs the split between tasks and runs.
- [ADR-003: Use Queue-First TaskRun Lifecycle with Central TaskManager Authority](../adrs/adr-003.md) — Governs run-state authority and queue-first transitions.

## Deliverables
- Manager-owned run lifecycle implementation with guarded transitions.
- Reconciliation of canonical task status from dependencies and runs.
- Propagated cancellation flow across parent tasks, subtasks, queued runs, and active runs.
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration tests for lifecycle and cancellation behavior **(REQUIRED)**

## Tests
- Unit tests:
  - [ ] Verify invalid run transitions such as `queued -> completed` and `running -> claimed` are rejected.
  - [ ] Verify task reconciliation moves tasks into `blocked`, `ready`, `in_progress`, and terminal states only from valid inputs.
  - [ ] Verify parent cancellation cancels queued descendant runs immediately and marks active descendant runs for cooperative stop.
- Integration tests:
  - [ ] Verify a queued run can progress through claim, start, complete, and task reconciliation against real storage.
  - [ ] Verify cancelling a task tree records cancellation events for the parent task, descendant tasks, and affected runs.
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- `TaskRun` state transitions are centralized and auditable
- Cancellation propagates predictably across task trees without leaving orphaned queue state
