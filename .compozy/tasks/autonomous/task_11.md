---
status: completed
title: Mechanical Scheduler Sweep Notify
type: backend
complexity: high
dependencies:
  - task_03
  - task_04
  - task_08
  - task_09
  - task_10
---

# Task 11: Mechanical Scheduler Sweep Notify

## Overview
Add the daemon-owned mechanical scheduler that wakes idle agents, sweeps expired leases, and rebuilds ephemeral scheduling state on boot. It provides safety and liveness mechanics only; it does not semantically decompose work and it never bypasses `ClaimNextRun`.

<critical>
- ALWAYS READ `_techspec.md`, ADR-003, ADR-004, ADR-009, and ADR-010 before adding scheduler behavior
- SCHEDULER MUST NOT CLAIM RUNS DIRECTLY - only sessions/operators call `ClaimNextRun`
- SCHEDULER STATE MUST BE REBUILDABLE from durable task/session data
- NO `scheduler.*` HOOK FAMILY IN MVP - use metrics/logs unless an external policy boundary exists
- TESTS REQUIRED - wake, sweep, boot recovery, idle-session selection, and shutdown behavior must be covered
- NO WORKAROUNDS - no fire-and-forget goroutines and no `time.Sleep` orchestration
</critical>

<requirements>
- MUST add a daemon-owned scheduler loop with context-bound lifecycle and explicit shutdown.
- MUST maintain only ephemeral/rebuildable wake state derived from durable task runs and sessions.
- MUST notify idle eligible sessions when pending work exists; sessions still claim through task_09/task_08.
- MUST sweep expired leases through the task service and record structured metrics/logs.
- MUST rebuild state on daemon boot before agents can observe stale pending/leased work.
- MUST expose minimal observability needed for QA without adding broad scheduler dashboards.
</requirements>

## Subtasks
- [x] 11.1 Add scheduler package or daemon-owned component with context, ticker/clock, and shutdown ownership.
- [x] 11.2 Implement pending-work scan, idle-session eligibility, and wake notification behavior.
- [x] 11.3 Implement expired-lease sweep through task service recovery APIs.
- [x] 11.4 Wire boot rebuild ordering in daemon startup before agent claim loops begin.
- [x] 11.5 Add metrics/logging for no-match, wake, sweep, and recovery outcomes without new hook families.
- [x] 11.6 Add deterministic lifecycle, recovery, and shutdown tests.

## Implementation Details
The scheduler is a mechanical safety layer. It should wake agents or coordinator sessions, not decide task decomposition. Keep responsibilities narrow enough that task_14 can add coordinator semantics on top without changing scheduler ownership.

Use test clocks or injected tick channels. Shutdown tests must prove goroutines exit through context cancellation and wait groups, not by hoping the process ends.

### Relevant Files
- `internal/daemon/daemon.go` - startup and shutdown composition.
- `internal/daemon/task_runtime.go` - current task runtime and boot recovery wiring.
- `internal/session/manager.go` - session lookup and wake/send behavior.
- `internal/task/manager.go` - pending scan and expired lease recovery APIs.
- `internal/observe/*` - metrics/log query surface if already available.
- `internal/hooks/*` - confirm no scheduler hook family is added in MVP.
- `.resources/hermes/cron/scheduler.py` - reference for scheduler/recovery loops.
- `.resources/hermes/environments/agent_loop.py` - reference for agent loop wake/claim separation.
- `.resources/multica/packages/core/inbox/ws-updaters.ts` - reference for wake/update notification flow.

### Dependent Files
- `internal/daemon/*coordinator*` - task_14 uses scheduler wake behavior for coordinator availability.
- `.compozy/tasks/autonomous/qa/test-cases/` - task_17 plans restart and recovery QA.

### Related ADRs
- [ADR-003: Task Run Claim Lease Model](adrs/adr-003.md) - lease sweep rules.
- [ADR-004: Coordinator-Agent Plus Mechanical Scheduler](adrs/adr-004.md) - semantic vs mechanical split.
- [ADR-009: Autonomy Hooks And Extension Contracts](adrs/adr-009.md) - no scheduler hook family in MVP.
- [ADR-010: Manual Operator Control Remains First-Class](adrs/adr-010.md) - scheduler supports manual and autonomous runs equally.

## Deliverables
- Context-owned mechanical scheduler wired into daemon startup/shutdown.
- Pending work wake notifications for eligible idle sessions.
- Expired lease sweep and boot rebuild behavior.
- Unit tests with 80%+ coverage for scheduler decision helpers **(REQUIRED)**.
- Integration tests for wake/sweep/restart behavior **(REQUIRED)**.

## Tests
- Unit tests:
  - [x] Scheduler selects only eligible idle sessions for pending work and skips busy sessions with active leases.
  - [x] Scheduler logs/metrics no-match cases without mutating task ownership.
  - [x] Expired lease sweep calls the task service recovery path and never writes task state directly.
  - [x] Boot rebuild derives state from durable task/session stores and ignores stale ephemeral entries.
  - [x] Shutdown cancels loops and waits for goroutines without leaks.
- Integration tests:
  - [x] A pending run wakes an idle eligible session, and the session claims it through `ClaimNextRun`.
  - [x] An expired lease is recovered and becomes claimable after daemon restart.
  - [x] Scheduler does not claim work when no session is eligible.
  - [x] Manual operator-started task runs are swept/woken the same way as agent-created runs.
  - [x] No `scheduler.*` hook events are emitted in MVP.
- Test coverage target: >=80%.
- All tests must pass.

## Success Criteria
- All tests passing.
- Test coverage >=80%.
- Scheduler improves liveness and recovery without becoming an orchestrator.
- All durable task ownership remains controlled by `ClaimNextRun` and lease fencing.
