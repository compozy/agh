## TC-INT-004: Durable Automation Scheduler Restart Safety

**Priority:** P0 (Critical)
**Type:** Integration
**Status:** Not Run
**Estimated Time:** 60 minutes
**Created:** 2026-04-25
**Last Updated:** 2026-04-25

### Objective

Verify durable automation scheduler invariants: cursor advancement before dispatch, at-most-once restart behavior, `skip_missed` boot reconciliation, delivery-error separation, and visibility through CLI/API/web/docs.

### Traceability

- Task: task_04, Durable Automation Scheduler.
- TechSpec: issues 20, 21, 22, and 25; Testing Approach cursor advancement before dispatch, restart reconciliation, duplicate-fire prevention, delivery-error persistence.
- ADR: ADR-002 durable scheduler state and at-most-once dispatch.
- Surfaces: `internal/automation`, `internal/store/globaldb`, `internal/api/contract/automation.go`, `internal/api/core/automation.go`, `internal/cli/automation.go`, web automation panels, site automation docs.

### Preconditions

- Isolated global DB with automation job fixtures.
- Controlled clock or deterministic scheduler test harness.
- Dispatcher can simulate success, delivery failure before agent handoff, daemon restart after cursor advancement, and missed fires.

### Test Steps

1. Register a scheduled job and advance the clock to a due fire.
   - **Expected:** Scheduler persists `next_run_at`, `last_scheduled_at`, and `last_fire_id` before dispatching work.

2. Simulate daemon stop after cursor advancement but before dispatch completion, then restart scheduler recovery.
   - **Expected:** The already claimed `last_fire_id` is not dispatched a second time and run count remains at most one for that fire.

3. Simulate daemon downtime past one or more due times with `skip_missed`.
   - **Expected:** Boot reconciliation records a misfire, increments/updates misfire diagnostics, and advances to the next future fire without dispatching stale work.

4. Simulate dispatch delivery failure.
   - **Expected:** `delivery_error` and `delivery_error_at` are recorded on the run while scheduler cursor state remains advanced and normal run execution error is not overwritten.

5. Query CLI and API job/run state.
   - **Expected:** `agh automation jobs get`, `agh automation runs get`, observe health, and HTTP payloads agree on scheduler cursor, fire ID, catch-up policy, misfire count, and delivery error fields.

6. Verify web and docs surfaces.
   - **Expected:** Web automation detail/run history renders scheduler and delivery-error fields; site automation and observe health docs describe the operator-visible fields.

### Evidence To Capture

- `qa/logs/TC-INT-004/go-test-automation.log`
- `qa/logs/TC-INT-004/job-before-restart.json`
- `qa/logs/TC-INT-004/job-after-restart.json`
- `qa/logs/TC-INT-004/run-delivery-error.json`
- `qa/screenshots/TC-INT-004/automation-detail-desktop.png` if browser validation is executed

### Edge Cases And Variations

| Variation | Input | Expected Result |
|-----------|-------|-----------------|
| Restart after cursor advance | Stop before dispatch completion | No duplicate fire |
| Missed cron run | Downtime past due time | Misfire recorded, next future cursor |
| Delivery failure | Dispatcher unavailable | Delivery error field set, cursor unchanged |
| Disabled job | `enabled=false` | Stored but not registered with scheduler |

### Related Test Cases

- TC-UI-001: Web automation rendering.
- TC-REG-002: Site automation documentation.
