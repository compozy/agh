---
status: pending
title: "Integrate automation with task-backed work items"
type: backend
complexity: high
dependencies:
  - task_05
  - task_06
---

# Task 10: Integrate automation with task-backed work items

## Overview
Integrate the new task domain with `internal/automation` without turning tasks into the universal execution engine. This task should let automation create task-backed work explicitly, either directly or through agents, while preserving non-task automation jobs and eliminating duplicated execution state for work that has already entered the task domain.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
1. Automation MUST support both accepted task-creation paths: direct task-service writes and agent-mediated explicit `task.create` flows.
2. Automation jobs that are not task-backed MUST continue to run through the existing automation runtime without being forced into the task domain.
3. Once work is explicitly materialized as task-backed, automation MUST not maintain a parallel execution state machine for that same unit of work.
</requirements>

## Subtasks
- [ ] 10.1 Define the automation-to-task integration seam and explicit task-backed job behavior.
- [ ] 10.2 Add direct automation-originated task creation and enqueue flows with trusted origin metadata.
- [ ] 10.3 Add support for agent-mediated task creation inside automation-driven sessions.
- [ ] 10.4 Prevent duplicate execution tracking between automation runs and task runs for task-backed work.
- [ ] 10.5 Carry workspace, channel, ownership, and idempotency context through automation-originated task flows.

## Implementation Details
Use the TechSpec "Integration Points" section for automation and the revised review outcome that tasks are explicit resources, not the universal wrapper for all work. Follow the patterns in `internal/automation/manager.go`, `dispatch.go`, and `types.go` to keep the integration daemon-owned and explicit.

### Relevant Files
- `internal/automation/manager.go` — Primary automation runtime entrypoint and composition surface.
- `internal/automation/dispatch.go` — Existing dispatch execution path that task-backed work must not duplicate.
- `internal/automation/types.go` — Existing run/job metadata model that will need task-aware integration points.
- `internal/daemon/boot.go` — Daemon composition root where the task manager and automation manager are wired together.
- `internal/task/` — Manager/service methods that automation will call.

### Dependent Files
- `internal/api/core/automation.go` — May need to surface task-backed automation behavior through shared handlers.
- `internal/observe/` — Will later observe automation-originated task events and run metrics.

### Related ADRs
- [ADR-003: Use Queue-First TaskRun Lifecycle with Central TaskManager Authority](../adrs/adr-003.md) — Governs manager-owned execution state once work becomes task-backed.
- [ADR-005: Derive Actor Identity Server-Side and Allow Optional Mutable Ownership](../adrs/adr-005.md) — Governs direct automation-originated actor and origin semantics.
- [ADR-006: Execute Subtasks Through an Injected Session Bridge with Dedicated Sessions by Default](../adrs/adr-006.md) — Governs agent-mediated session-backed task execution.

## Deliverables
- Explicit automation-to-task integration for direct and agent-mediated task creation.
- Non-overlap rules preventing duplicated execution state between automation runs and task runs.
- Tests covering task-backed and non-task automation behavior.
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration tests for automation-originated task flows **(REQUIRED)**

## Tests
- Unit tests:
  - [ ] Verify direct automation-originated task creation derives immutable origin and creator metadata server-side.
  - [ ] Verify non-task automation jobs continue to use the existing automation runtime unchanged.
  - [ ] Verify task-backed automation dispatch refuses to maintain a second execution state machine for the same work item.
- Integration tests:
  - [ ] Verify an automation job can create a task directly and enqueue a task run with the expected origin and idempotency metadata.
  - [ ] Verify an automation-launched agent session can explicitly call `task.create` and produce a task whose `created_by` is the agent while `origin` remains automation-linked.
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- Automation can create task-backed work explicitly without forcing all automation work into tasks
- There is only one canonical execution lifecycle for work once it becomes task-backed
