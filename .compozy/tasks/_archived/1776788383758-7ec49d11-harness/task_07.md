---
status: completed
title: Task-run completion to synthetic reentry bridge
type: backend
complexity: critical
dependencies:
  - task_04
  - task_06
---

# Task 07: Task-run completion to synthetic reentry bridge

## Overview

Connect detached harness task-run completion to policy-based synthetic session reentry. This task is the cross-cutting bridge that decides when a completed detached run should wake its owning session, how that wake-up is queued, and how completion remains observable even when the policy chooses silent completion instead of reentry.

<critical>
- ALWAYS READ `_techspec.md`, ADRs, `task_04.md`, and `task_06.md` before starting
- REFERENCE TECHSPEC sections "Workstream 4: Synthetic Reentry Model", "Workstream 5: Detached Async Runtime on Task Infrastructure", and "Required Behavior"
- FOCUS ON "WHAT" - bridge task-run completion into synthetic wake-up policy; do not reimplement task runtime lifecycle or transcript consumers here
- MINIMIZE CODE - one daemon-owned completion bridge, not per-caller ad hoc wake-up logic
- TESTS REQUIRED - wake-up vs silent completion policy, ordering, shutdown behavior, and recovery all need coverage
- GREENFIELD: reentry precisa ser push-driven e auditavel; nao cair em polling escondido ou callback ad hoc
</critical>

<requirements>
- MUST observe detached harness task-run completion and apply policy to decide wake-up versus silent completion
- MUST create synthetic prompt submissions only through the dedicated synthetic path introduced earlier
- MUST keep completion observable even when no wake-up is emitted
- MUST define queueing behavior for completions that arrive while the target session is busy or unavailable
- MUST preserve shutdown and boot-recovery behavior without duplicating task-runtime reconciliation logic
</requirements>

## Subtasks
- [x] 7.1 Add the daemon-owned completion bridge from task-run terminal states into harness wake-up decisions
- [x] 7.2 Implement wake-up policy evaluation for reenter versus silent completion behavior
- [x] 7.3 Queue or drop synthetic wake-ups according to session state and explicit runtime rules
- [x] 7.4 Emit observable completion and wake-up signals regardless of whether reentry occurs
- [x] 7.5 Add integration coverage for completion, wake-up, shutdown, and recovery scenarios

## Implementation Details

See TechSpec "Workstream 4: Synthetic Reentry Model", "Workstream 5: Detached Async Runtime on Task Infrastructure", and ADR-003. This is the core orchestration bridge that turns durable detached execution into actual harness behavior without confusing the task runtime with transcript or prompt semantics.

### Relevant Files
- `internal/daemon/task_runtime.go` - detached run completion and recovery already live here and need a harness-owned bridge
- `internal/task/manager.go` - task-run completion semantics originate here and must remain authoritative
- `internal/session/manager_prompt.go` - synthetic prompt submission path from task_04 is the only legal reentry path
- `internal/store/globaldb/global_db_task.go` - run persistence and terminal-state inspection must remain durable and queryable
- `internal/observe/observer.go` - completion and wake-up visibility should surface through existing observability seams
- `internal/daemon/harness_reentry_bridge.go` - new daemon-owned bridge module introduced by this task

### Dependent Files
- `internal/daemon/task_runtime_test.go` - should gain coverage for wake-up versus silent completion behavior
- `internal/daemon/daemon_automation_task_integration_test.go` - existing detached-runtime lane is a natural place for synthetic reentry integration coverage
- `internal/session/manager_test.go` - synthetic prompt submission semantics may need to be asserted under queued reentry scenarios
- `internal/store/globaldb/global_db_task_integration_test.go` - terminal-state persistence plus wake-up metadata may need cross-store assertions

### Related ADRs
- [ADR-001: Resolve Harness Behavior from Durable Session Context and Turn Origin](adrs/adr-001.md) - Wake-up policy is evaluated on top of the resolved harness model
- [ADR-003: Reuse the Task Runtime for Detached Harness Work and Policy-Based Synthetic Reentry](adrs/adr-003.md) - This task implements the policy-based reentry half of the ADR

### External References
- `.resources/claude-code/utils/task/framework.ts` - strong reference for task registration, polling, and completion event emission
- `.resources/claude-code/tasks/LocalMainSessionTask.ts` - useful precedent for background completion surfacing back to a foreground session
- `.resources/claude-code/utils/sdkEventQueue.ts` - helpful reference for internal event queues tied to task lifecycle
- `.resources/openclaw/src/agents/subagent-registry-run-manager.ts` - strong analog for waiting, completion, and post-run cleanup/reentry behavior
- `.resources/hermes/tools/process_registry.py` - useful completion-queue and restart-recovery model for detached work
- `.resources/hermes/gateway/run.py` - shows internal message injection after background/runtime events
- `.resources/openfang/crates/openfang-kernel/src/background.rs` - helpful for background completion orchestration rules
- `.resources/openfang/crates/openfang-kernel/src/event_bus.rs` - useful reference for observable fan-out on completion and wake-up

## Deliverables
- Daemon-owned bridge from detached task-run completion into synthetic reentry policy
- Explicit wake-up versus silent-completion behavior **(REQUIRED)**
- Queue/drop behavior for busy or unavailable target sessions **(REQUIRED)**
- Observable completion and wake-up signals for every detached harness run **(REQUIRED)**
- Idempotency and duplicate-completion regression coverage for the reentry bridge **(REQUIRED)**
- Unit and integration tests with >=80% coverage for the completion bridge and affected runtime paths **(REQUIRED)**

## Tests
- Unit tests:
  - [x] Completed detached runs trigger the expected wake-up policy decision from their persisted harness metadata
  - [x] Silent-completion policy records observability without enqueuing or dispatching a synthetic prompt
  - [x] Busy-session behavior queues one synthetic wake-up instead of bypassing the current active turn
  - [x] Missing, stopped, or not-yet-resumable target sessions are handled according to the explicit drop/report rules
  - [x] Duplicate terminal notifications for the same `task_run` do not emit duplicate synthetic wake-ups
- Integration tests:
  - [x] Detached run completion can wake a live session through the synthetic prompt path end to end, including persisted event creation
  - [x] Multiple completed runs targeting the same session are reentered in deterministic FIFO order
  - [x] Shutdown or boot-recovery scenarios do not emit duplicate wake-ups for the same completed run
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- Detached task-run completion can reenter sessions through one daemon-owned bridge
- AGH records completion observably whether or not a given run wakes a session
