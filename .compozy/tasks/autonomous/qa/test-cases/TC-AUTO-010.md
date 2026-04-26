## TC-AUTO-010: Mechanical Scheduler Wake, Sweep, And Restart Recovery

**Priority:** P0 (Critical)
**Type:** Integration
**Status:** Not Run
**Estimated Time:** 55 minutes
**Created:** 2026-04-26
**Last Updated:** 2026-04-26

### Objective

Verify the daemon-owned scheduler is a context-owned wake/sweep/recovery component that rebuilds
from durable state, wakes eligible idle sessions, recovers expired leases through task service APIs,
and never claims work directly.

### Traceability

- Task: task_11, Mechanical Scheduler Sweep Notify.
- TechSpec: Mechanical Scheduler, Scheduler and Claim Authority, Monitoring and Observability.
- ADR: ADR-003, ADR-004, ADR-009, ADR-010.
- Resource lesson: Hermes scheduler reference emphasizes lock/recovery discipline and delivery-error separation from ownership authority.
- Surfaces: `internal/scheduler`, `internal/daemon`, `internal/task`, session wake/notify path, logs/metrics.

### Preconditions

- Daemon can start with isolated global DB and deterministic scheduler interval/test clock.
- At least one idle eligible session and one queued run with matching capabilities.
- At least one expired active lease for recovery.

### Test Steps

1. Start daemon and let scheduler rebuild state from durable sessions/runs.
   - **Expected:** Rebuild completes before claim traffic is accepted; stale ephemeral scheduler state is ignored.

2. Enqueue a pending run with one eligible idle session.
   - **Expected:** Scheduler wakes/notifies the session, but ownership remains unclaimed until the session calls `ClaimNextRun` or `agh task next`.

3. Let the eligible session claim through public task API.
   - **Expected:** Claim state is written by task service and not by scheduler internals.

4. Seed an expired lease and run scheduler sweep.
   - **Expected:** Scheduler calls task service recovery; run becomes claimable or finalizes according to policy, and stale holder tokens fail.

5. Stop daemon.
   - **Expected:** Scheduler goroutine exits by context cancellation/wait group, not process exit or sleep timing.

6. Query hooks/events for scheduler families.
   - **Expected:** No `scheduler.*` hooks are emitted in the MVP; only metrics/logs/observability exist.

### Evidence To Capture

- `qa/logs/TC-AUTO-010/scheduler-boot-rebuild.log`
- `qa/logs/TC-AUTO-010/scheduler-wake.log`
- `qa/logs/TC-AUTO-010/claim-after-wake.json`
- `qa/logs/TC-AUTO-010/expired-sweep.log`
- `qa/logs/TC-AUTO-010/scheduler-shutdown.log`

### Edge Cases And Variations

| Variation | Input | Expected Result |
|-----------|-------|-----------------|
| No eligible session | queued run but no matching capability | Logs/metrics no-match, no ownership mutation |
| Busy session | active lease present | Session skipped for wake |
| Daemon restart | expired lease persisted | Boot recovery makes run claimable |
| Hook catalog | `scheduler.*` search | No scheduler hook family |

### Related Test Cases

- TC-AUTO-007: Lease recovery correctness.
- TC-AUTO-013: Coordinator integrates with scheduler wake behavior.
