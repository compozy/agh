## TC-AUTO-003: Autonomy Hook Taxonomy And Safety Guards

**Priority:** P1 (High)
**Type:** Integration
**Status:** Not Run
**Estimated Time:** 40 minutes
**Created:** 2026-04-26
**Last Updated:** 2026-04-26

### Objective

Verify the autonomy hook families are first-class typed contracts and that hook patches cannot
weaken daemon-owned safety boundaries for task claims, leases, coordinator bootstrap, or safe spawn.

### Traceability

- Task: task_03, Autonomy Hook Taxonomy And Task Hook Bridge.
- TechSpec: Autonomy Hook and Resource Surface, Domain events vs hooks bridge, Hook payloads.
- ADR: ADR-004, ADR-009, ADR-012.
- Resource lesson: Hermes shell hook references favor explicit hook invocation around runtime actions instead of table tailing.
- Surfaces: `internal/hooks`, `internal/task` hook dispatcher, `internal/daemon/hooks_bridge.go`, hook binding resources.

### Preconditions

- Hook declarations can be registered from config or hook binding resources in an isolated workspace.
- Test hook executors can return deny/narrow patches and forbidden widening patches.

### Test Steps

1. Inspect `agh hooks events -o json` or equivalent hook catalog output.
   - **Expected:** `coordinator.*`, `task.run.*`, and `spawn.*` events are listed with payload, patch, family, and sync eligibility metadata; no `scheduler.*` family exists.

2. Register an observation hook for `task.run.enqueued` and enqueue a run.
   - **Expected:** The task audit event commits first, then the typed hook payload includes `task_id`, `run_id`, actor/origin, and `coordination_channel_id` when present.

3. Register a `task.run.pre_claim` hook that denies or narrows criteria.
   - **Expected:** Denial prevents claim before transaction commit; narrowing may add required capabilities or raise priority only.

4. Register a malicious `task.run.pre_claim` hook that removes requirements or changes claimant identity.
   - **Expected:** Runtime rejects the patch and no run ownership is mutated.

5. Register a `spawn.pre_create` hook that tries to widen child permissions or create a coordinator role.
   - **Expected:** Daemon rejects the patched request after hook processing; unknown child atoms count as widening.

### Evidence To Capture

- `qa/logs/TC-AUTO-003/hooks-events.json`
- `qa/logs/TC-AUTO-003/task-run-hook-dispatch.log`
- `qa/logs/TC-AUTO-003/pre-claim-deny.log`
- `qa/logs/TC-AUTO-003/forbidden-patch-rejection.log`

### Edge Cases And Variations

| Variation | Input | Expected Result |
|-----------|-------|-----------------|
| Async observation | `task.run.post_claim` | Records payload without mutating committed claim |
| Missing channel | non-coordinated run | Payload omits channel without fabricating data |
| Hook timeout | required sync hook times out | Operation fails according to hook policy |
| Scheduler event query | `scheduler.*` | No scheduler hook family is present |

### Related Test Cases

- TC-AUTO-007: Claim transaction safety.
- TC-AUTO-012: Spawn safety after hook patches.
