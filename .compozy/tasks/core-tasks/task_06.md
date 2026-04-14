---
status: pending
title: "Wire the session bridge, dedicated subtask sessions, and boot recovery"
type: backend
complexity: critical
dependencies:
  - task_01
  - task_05
---

# Task 06: Wire the session bridge, dedicated subtask sessions, and boot recovery

## Overview
Connect task execution to the existing session runtime through the injected bridge accepted in the TechSpec. This task makes executable subtasks start in dedicated sessions by default, preserves explicit attach flows for resume and handoff cases, and closes the cold-start gap for orphaned runs after daemon restarts.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
1. The task domain MUST interact with session execution only through the injected bridge defined in `internal/task`, not by importing `internal/session`.
2. Starting an executable subtask MUST allocate a dedicated new session by default, while session attach MUST remain an explicit and state-gated path.
3. Daemon boot MUST reconcile orphaned or in-flight task runs against live session state and move them to the correct post-restart state.
</requirements>

## Subtasks
- [ ] 6.1 Implement the bridge adapter between `internal/task` and the existing session manager.
- [ ] 6.2 Wire dedicated-session allocation for executable subtasks started through `TaskRun`.
- [ ] 6.3 Implement explicit attach-session flows with single-assignment and lifecycle gating.
- [ ] 6.4 Add daemon boot reconciliation for orphaned or stale task runs after restart.
- [ ] 6.5 Ensure stop requests from task cancellation use the cooperative-then-forced shutdown path.

## Implementation Details
Use the TechSpec sections "Run Authority and Attachment Rules", "Cancellation Model", "Cold-Start Recovery", and the ADR for the injected session bridge. The daemon remains the composition root and should own bridge construction and boot-time recovery orchestration.

### Relevant Files
- `internal/session/interfaces.go` — Reference the session manager surface that the task bridge must adapt to.
- `internal/session/manager_start.go` — Reference session creation and startup behavior that dedicated-subtask sessions must reuse.
- `internal/session/manager_lifecycle.go` — Reference stop and lifecycle semantics used during task cancellation.
- `internal/daemon/boot.go` — Boot-time composition and reconciliation entrypoint.
- `internal/daemon/orphan.go` — Existing orphan cleanup patterns relevant to task-run cold-start recovery.
- `internal/daemon/boundary.go` — Reference daemon-owned boundary and injection patterns.
- `internal/task/` — New bridge interface and task-side lifecycle code to connect here.

### Dependent Files
- `internal/api/core/handlers.go` — Lifecycle handlers will depend on the bridge-backed run start flows.
- `internal/automation/dispatch.go` — Task-backed automation flows will depend on the dedicated-session behavior introduced here.
- `internal/extension/host_api.go` — Extension-originated subtask runs will depend on this bridge behavior.

### Related ADRs
- [ADR-003: Use Queue-First TaskRun Lifecycle with Central TaskManager Authority](../adrs/adr-003.md) — Governs queue-first run progression and cold-start ownership.
- [ADR-006: Execute Subtasks Through an Injected Session Bridge with Dedicated Sessions by Default](../adrs/adr-006.md) — Governs the bridge seam and dedicated-session default.

## Deliverables
- A daemon-wired session bridge implementation satisfying the `internal/task` interface.
- Dedicated-session allocation for executable subtasks and explicit attach-session support.
- Boot-time recovery for orphaned or in-flight task runs.
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration tests for session-backed task execution and boot recovery **(REQUIRED)**

## Tests
- Unit tests:
  - [ ] Verify attach-session rejects attempts after a run is already bound to another session.
  - [ ] Verify run start requests choose dedicated session allocation when no explicit attach target is supplied.
  - [ ] Verify cold-start reconciliation classifies `claimed`, `starting`, and `running` runs correctly when their sessions are missing or stopped.
- Integration tests:
  - [ ] Verify starting an executable subtask creates a dedicated session and persists the attached `session_id`.
  - [ ] Verify daemon restart reclassifies orphaned in-flight runs and records recovery audit events.
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- Executable subtasks consistently run in dedicated sessions unless explicit attach is requested
- Boot-time recovery prevents stale in-flight task runs from surviving daemon restarts incorrectly
