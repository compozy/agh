# Test Specification: Scheduler Capacity-Aware Starvation

Canonical test contract for the scheduler capacity-aware starvation correction. Companion to [`_techspec.md`](_techspec.md).

No `_user_stories.md` exists for this incident. Coverage derives from Marketplace Task 11, the Northstar capacity-saturation reproduction, ADR-001, and every component/interface in the TechSpec. Stable IDs in this document must not be renumbered after implementation begins.

## Strategy

- **Frameworks and harnesses:** Go `testing`, `testify` only where already canonical, `clockwork` fake time, existing scheduler/session/store fakes at I/O boundaries, real SQLite through `globaldb` integration fixtures, and the existing Northstar real-scenario harness.
- **Execution:** focused suites run with `CGO_ENABLED=1` and `-race`; frontend commands are unnecessary unless a public-shape drift appears because this change modifies no Web or generated contract source.
- **Conventions:** extend the named canonical suites, use `t.Run("Should …")`, use `t.Parallel()` unless shared global/process state forbids it, handle every error, and assert observable state rather than implementation literals.
- **Test placement invariant:** capacity classification belongs to `internal/scheduler`; session projection belongs to `internal/daemon`; durable pressure reads belong to `internal/store/globaldb`; public composition belongs to `internal/task`; serial task-role reuse belongs to `internal/daemon`; release-grade behavior belongs to the existing Northstar QA journey.

## Coverage Matrix

| Source | Behavior | Unit | Integration | E2E |
| --- | --- | --- | --- | --- |
| QA-INC-001 | Busy compatible owner-pool capacity does not become false starvation | UT-002, UT-003, UT-004, UT-014 | IT-001, IT-002 | E2E-001 |
| QA-INC-001.EC-1 | Same-cycle reservation holds later serial backlog | UT-005, UT-026 | IT-001 | E2E-001 |
| QA-INC-001.EC-2 | Capability projection uncertainty cannot prove absence | UT-012, UT-023 | IT-009 | — |
| ADR-001 | Policy A preserves serial waiting and rejects elastic expansion | UT-014, UT-015, UT-024 | IT-001, IT-008 | E2E-003 |
| Capacity classifier | Exhaustive available/waiting/unmatched/indeterminate disposition | UT-001–UT-012 | IT-007, IT-009 | — |
| Scheduler selection | Wake only available sessions and route only true candidates to convergence | UT-006, UT-013–UT-018, UT-026 | IT-001–IT-004 | E2E-001 |
| Durable convergence | Freeze/resume existing budget without duplicate side effects | UT-015–UT-018 | IT-003, IT-006 | — |
| Daemon session projection | Preserve live session identity and capability certainty | UT-022, UT-023 | IT-009 | — |
| Public scheduler status | Count active episodes and preserve pause/status parity | UT-019–UT-021, UT-025 | IT-005, IT-010 | E2E-002 |
| Task-role runtime | Reuse one role session and drain serial work | UT-027 | IT-008 | E2E-001, E2E-003 |
| Workspace isolation | Capacity from another scope/channel/owner cannot hold a run | UT-008–UT-011 | IT-007 | E2E-001 |
| Observability | Capacity wait is distinct from no-match/starved signals | UT-024, UT-025 | IT-001 | E2E-002 |

## Unit Tests

### Capacity Classifier (`internal/scheduler/scheduler_test.go`)

Extend `TestRunOnceEscalatesStarvedRuns` and the existing owner/scope suites; do not create a duplicate top-level regression owner.

- **UT-001** (happy): `classifyRunCapacity` receives one active, idle, unreserved session matching workspace `ws-1`, channel `frontend`, pool owner `frontend-agent`, and required capability `frontend`; it returns `CapacityAvailable` with that session as the sole available candidate.
- **UT-002** (state): the same session owns an active `running` task-run lease; `classifyRunCapacity` returns `CapacityWaiting`.
- **UT-003** (state): the same active session reports `Prompting=true`; `classifyRunCapacity` returns `CapacityWaiting`.
- **UT-004** (state): a matching task-role session is in `starting`; `classifyRunCapacity` returns `CapacityWaiting`.
- **UT-005** (ordering): a matching idle session is already present in the same-cycle occupied/reserved set; `classifyRunCapacity` returns `CapacityWaiting` for the later run.
- **UT-006** (happy): one matching session is busy and a second matching session is idle; `classifyRunCapacity` returns `CapacityAvailable` containing only the idle session.
- **UT-007** (state): no live session matches an ownerless workspace run with required capability `go`; `classifyRunCapacity` returns `CapacityUnmatched`.
- **UT-008** (state): the only otherwise-compatible session belongs to `ws-2` while the task belongs to `ws-1`; `classifyRunCapacity` returns `CapacityUnmatched`.
- **UT-009** (state): the only otherwise-compatible session is in channel `backend` while the run requires `frontend`; `classifyRunCapacity` returns `CapacityUnmatched`.
- **UT-010** (state): the only session has agent name `analytics-agent` while pool owner is `frontend-agent`; `classifyRunCapacity` returns `CapacityUnmatched`.
- **UT-011** (state): the only known-capability session lacks required capability `frontend`; `classifyRunCapacity` returns `CapacityUnmatched`.
- **UT-012** (error): a live session matches workspace, channel, and owner but has `CapabilityStateUnknown` for a run requiring `frontend`; `classifyRunCapacity` returns `CapacityIndeterminate`.

### Selection And Convergence (`internal/scheduler/scheduler_test.go`)

- **UT-013** (state): an available session is repeatedly woken while the run remains queued past `MinQueuedAge`; the scheduler creates or advances exactly one `task_run_starvation` episode for that cycle.
- **UT-014** (state): a run older than `MinQueuedAge` is `CapacityWaiting`; one `RunOnce` reports it held and does not create a starvation row.
- **UT-015** (idempotency): a `CapacityWaiting` run already has `wake_count=3`; one `RunOnce` leaves `wake_count=3` and all episode timestamps unchanged.
- **UT-016** (state): the held session becomes idle on the next cycle; the scheduler selects that session for an advisory wake.
- **UT-017** (state): the held session disappears on the next cycle while the run remains old and queued; the scheduler classifies the run unmatched and advances its frozen budget from 3 to 4.
- **UT-018** (boundary): a run becomes available exactly at `max(queued_at, scheduler_pause.updated_at) + MinQueuedAge`; convergence can advance once, while one nanosecond before that boundary it cannot.

### Durable/Public Pressure (`internal/store/globaldb/global_db_task_starvation_test.go` and `internal/task/scheduler_controls_test.go`)

- **UT-019** (state): `CountEscalatingQueuedTaskRuns` sees an unpaused claimable queued run with an active `task_run_starvation` row; it returns 1.
- **UT-020** (boundary): an unpaused claimable queued run is older than two minutes but has no starvation row; `CountEscalatingQueuedTaskRuns` returns 0.
- **UT-021** (state): a starvation row belongs to a claimed, running, terminal, directly paused, or ancestor-paused run; each table case returns 0.
- **UT-022** (happy): `schedulerSessionSource.Sessions` receives a known empty capability projection; it returns the session with `CapabilityStateKnown` and an empty capability slice.
- **UT-023** (error): capability projection returns `context deadline exceeded` while the outer scheduler context remains live; `Sessions` returns the session with `CapabilityStateUnknown` and emits `scheduler.session_context.error` with its session ID.
- **UT-024** (state): a cycle holding two runs sets `CapacityWaitingRuns=2` and contains both run IDs without increasing `StarvedRuns` or `NoMatchRuns`.
- **UT-025** (state): scheduler `Stats` accumulates capacity-wait counts across cycles while leaving public `SchedulerStatus` fields unchanged.
- **UT-026** (ordering): two queued runs target one idle session in a cycle; priority and queued-time ordering select the canonical first run, while the later run is capacity waiting rather than unmatched.

### Task-Role Reuse (`internal/daemon/task_role_runtime_test.go`)

- **UT-027** (idempotency): `activateForStarvation` sees an existing reusable task-role session with the same agent, channel, scope, and execution-profile fingerprint; it creates no second session.

## Integration Tests

### Scheduler, Task Service, And Real Store

- **IT-001**: use a real queued run plus one matching session that owns another active run; execute at least 12 scheduler cycles beyond `MinQueuedAge`; assert no spawn request, `task.run_starved`, `task.run_needs_attention`, or run status transition occurs.
- **IT-002**: after IT-001, complete/release the active run so the session becomes idle; execute the next scheduler cycle; assert the waiting run is woken and remains claimable through `task.Service.ClaimNextRun`.
- **IT-003**: keep a compatible idle session from claiming across the configured ladder; assert wake fan-out, one spawn request, one `task.run_starved`, and final `needs_attention` occur at their configured tiers.
- **IT-004**: provide no structurally compatible live session; assert the existing unmatched convergence ladder and capability-matched spawn path remain operational.
- **IT-005**: persist one age-only queued run and one queued run with `task_run_starvation`; call the shared HTTP and UDS `GET /api/scheduler` handlers plus `agh scheduler status -o json`; assert all three report `starved_run_count=1` with identical payload semantics.
- **IT-006**: persist `wake_count=3`, restart/rebuild the scheduler with matching busy capacity, and execute a cycle; assert the row remains at 3, then remove capacity and assert the next eligible cycle advances to 4.
- **IT-007**: queue a `ws-1`/`frontend`/`frontend-agent` run while only a matching-looking `ws-2` or wrong-channel session is busy; assert foreign capacity does not hold the run and convergence treats it as unmatched.
- **IT-008**: activate three sibling runs for one task-role agent/channel/profile; assert exactly one reusable session is created and its synthetic work prompts drain serially without a second active lease.
- **IT-009**: make situation capability projection fail for an otherwise matching live session across multiple cycles; assert the run stays queued, no convergence side effect occurs, and a projection diagnostic is observable each cycle.
- **IT-010**: pause the scheduler with an existing starvation row, resume it, and query status before and at the grace boundary; assert count 0 before the boundary and count 1 at the boundary.

## End-to-End Tests

### Marketplace Northstar Real-Scenario QA

- **E2E-001**: bootstrap a fresh isolated Northstar lab, register only `<lab>/project`, create the declared agents/tasks, activate all runs behind scheduler pause, post exactly one confirmed PM kickoff, resume, and observe multiple frontend siblings drain serially without any compatible-capacity run entering `needs_attention`; finish with the strict playbook verdict required by Task 11.
- **E2E-002**: during E2E-001, capture `agh scheduler status -o json`, `GET /api/scheduler`, Tasks Web, and runtime logs while the frontend worker is busy; assert queued backlog and active claim are truthful, `starved_run_count` excludes capacity wait, and `scheduler.capacity_waiting` identifies held run IDs without secrets.
- **E2E-003**: during E2E-001, inspect session inventory and provider evidence; assert policy A creates no additional task-role session merely because a compatible session is occupied.

## Required Verification Commands

Run from the repository root with RTK:

```bash
rtk env CGO_ENABLED=1 go test -race ./internal/scheduler/... -count=1
rtk env CGO_ENABLED=1 go test -race ./internal/daemon/... -run 'TestScheduler|TestTaskRole' -count=1
rtk env CGO_ENABLED=1 go test -race ./internal/task/... ./internal/store/globaldb/... -count=1
rtk make lint
rtk git diff --check
```

The final Marketplace completion gate remains one fresh `rtk make verify` after Task 11 and deep review, not during this implementation step.
