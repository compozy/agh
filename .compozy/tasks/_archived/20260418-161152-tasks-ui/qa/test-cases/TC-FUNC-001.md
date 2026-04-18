## TC-FUNC-001: Tasks Dashboard aggregate health and active runs

**Priority:** P0
**Type:** Functional
**Status:** Not Run
**Estimated Time:** 12 minutes
**Created:** 2026-04-17
**Last Updated:** 2026-04-17
**Module:** Tasks Dashboard
**Route / Surface:** `/_app/tasks` in dashboard mode
**Design Reference:** `docs/design/paper/tasks/AGH Tasks — Dashboard@2x.png`
**Execution Lane:** Manual + browser regression

### Objective

Verify the Tasks dashboard renders the observer-backed aggregate read model with queue health, freshness, status breakdown, and active-run visibility that matches the planned operator workflow.

### Preconditions

- [ ] The daemon-served web app is running through the standard repo harness.
- [ ] A workspace is available or the global-workspace onboarding path can be used.
- [ ] Seed data includes at least one in-progress task, one blocked or failed task, and one active run.
- [ ] Dashboard queries are returning non-empty aggregate data.

### Test Steps

1. Open the app root and resolve onboarding if shown.
   **Expected:** The app shell loads and the Tasks sidebar entry is available.
2. Open Tasks and switch to dashboard mode.
   **Expected:** The dashboard view loads inside the shared Tasks shell and does not fall back to list/kanban layout.
3. Inspect the dashboard cards, queue-health section, freshness label, and active-runs area.
   **Expected:** Counts, warnings, backlog state, and freshness copy reflect the seeded observer payload.
4. Open one linked task or run from the dashboard if a link is present.
   **Expected:** Navigation reaches the matching task or run-detail context without losing workspace state.
5. Record a screenshot or browser artifact for the dashboard.
   **Expected:** Execution evidence can be stored under `.compozy/tasks/tasks-ui/qa/screenshots/`.

### Edge Cases & Variations

| Variation | Input / State | Expected Result |
| --- | --- | --- |
| Empty aggregate | no dashboard data | the dashboard renders a distinct empty state instead of a broken list shell |
| Warning state | backlog, stale freshness, or stuck run | warning copy and health indicators remain visible and understandable |
| No active runs | seeded aggregate without active runs | the active-runs section falls back cleanly without breaking the rest of the page |

### Related Test Cases

- `TC-FUNC-002`
- `TC-FUNC-003`
- `TC-FUNC-008`

### Notes

- This case is part of Smoke because task_19 requires at least one aggregate Tasks mode in the browser lane.
