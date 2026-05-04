## TC-UI-001: Web Automation Scheduler And Run Diagnostics

**Priority:** P1 (High)
**Type:** UI
**Status:** Not Run
**Estimated Time:** 35 minutes
**Created:** 2026-04-25
**Last Updated:** 2026-04-25

### Objective

Verify that the web automation experience renders durable scheduler state and delivery-error diagnostics from updated contracts without layout regressions.

### Traceability

- Task: task_04 plus task_10 extra automation coverage.
- TechSpec: issues 20, 21, 22, and 25.
- ADR: ADR-002 durable scheduler state.
- Surfaces: `web/src/systems/automation`, automation API adapter/tests/fixtures, automation detail panel, run history, generated OpenAPI types, site automation docs for expected fields.

### Preconditions

- Web dependencies are installed.
- Fixture job contains `scheduler.next_run_at`, `last_scheduled_at`, `last_fire_id`, `catch_up_policy`, `misfire_count`, and registered state.
- Fixture run contains `fire_id`, `scheduled_at`, `delivery_error`, and `delivery_error_at`.

### Test Steps

1. Run focused automation web tests.
   - **Expected:** Tests assert scheduler state and delivery-error fields render from contract fixtures.

2. Typecheck the web app.
   - **Expected:** Generated automation DTOs match adapter/component expectations.

3. Open or screenshot the automation detail panel at desktop, tablet, and mobile sizes if browser validation is part of task_11.
   - **Expected:** Scheduler fields are visible, text does not overflow, and no unrelated card nesting/layout shift appears.

4. Verify run history delivery error rendering.
   - **Expected:** Delivery diagnostic is distinct from normal run status/error and visible in the run list/details.

5. Cross-check docs.
   - **Expected:** Site docs describe the same fields the web UI exposes.

### Evidence To Capture

- `qa/logs/TC-UI-001/web-automation-vitest.log`
- `qa/logs/TC-UI-001/web-typecheck.log`
- `qa/screenshots/TC-UI-001/automation-desktop.png`
- `qa/screenshots/TC-UI-001/automation-mobile.png`

### Edge Cases And Variations

| Variation | Input | Expected Result |
|-----------|-------|-----------------|
| Scheduler missing | `job.scheduler` absent | UI shows idle/no scheduler state without crash |
| Long fire ID | Long `last_fire_id` | Text wraps/truncates cleanly |
| Delivery error present | `delivery_error` set | Delivery label appears separately |
| Zero misfires | `misfire_count=0` | Displays zero or neutral state correctly |

### Related Test Cases

- TC-INT-004: Backend automation invariant.
