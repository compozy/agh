## TC-FUNC-008: Task run-detail route and linked-session drill-down

**Priority:** P0
**Type:** Functional
**Status:** Not Run
**Estimated Time:** 12 minutes
**Created:** 2026-04-17
**Last Updated:** 2026-04-17
**Module:** Task Run Detail
**Route / Surface:** `/_app/tasks/$id/runs/$runId`
**Design Reference:** `docs/design/paper/tasks/AGH Tasks — Run Detail@2x.png`
**Execution Lane:** Manual + browser regression

### Objective

Verify run-detail deep links render task-run timing, progress, activity, and linked-session drill-down behavior from the task-native run-detail surface.

### Preconditions

- [ ] A seeded task run exists with a valid `runId`.
- [ ] The run-detail route is reachable from task detail or direct navigation.
- [ ] A linked session is present when the scenario is meant to prove drill-down.

### Test Steps

1. Open the run-detail route from task detail or direct URL.
   **Expected:** The run-detail layout renders without falling back to a generic task placeholder.
2. Inspect summary, timing, progress, and activity panels.
   **Expected:** Run metadata, execution state, and error or progress summaries match the seeded run payload.
3. Use the linked-session affordance when available.
   **Expected:** The operator can navigate into `/session/$id` from the run-detail surface.
4. Navigate back to task detail or the task list.
   **Expected:** The route transition preserves the operator's context within the Tasks route family.

### Edge Cases & Variations

| Variation | Input / State | Expected Result |
| --- | --- | --- |
| No linked session | run summary without session reference | the UI falls back cleanly and still renders the run detail |
| Loading or not found | stale or missing run ID | dedicated loading/not-found states appear instead of generic route failures |
| Failed run | run has error data | the error summary remains visible and actionable |

### Related Test Cases

- `TC-FUNC-001`
- `TC-FUNC-007`

### Notes

- This case is part of Smoke because task_19 requires run-detail browser coverage.
