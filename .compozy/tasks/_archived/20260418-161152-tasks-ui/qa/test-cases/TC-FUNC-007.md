## TC-FUNC-007: Task detail route timeline, tabs, and related panels

**Priority:** P0
**Type:** Functional
**Status:** Not Run
**Estimated Time:** 15 minutes
**Created:** 2026-04-17
**Last Updated:** 2026-04-17
**Module:** Task Detail
**Route / Surface:** `/_app/tasks/$id`
**Design Reference:** `docs/design/paper/tasks/AGH Tasks — Detail (Events SSE)@2x.png`
**Execution Lane:** Manual + browser regression

### Objective

Verify the task-detail route presents task-native detail, timeline, runs, dependencies, and related navigation without falling back to session-first composition.

### Preconditions

- [ ] A seeded task exists with timeline activity and at least one related dependency, child, or run reference.
- [ ] The task detail route is reachable directly or from the list view.

### Test Steps

1. Open a seeded task detail route directly or from the list preview.
   **Expected:** The detail header renders the correct identifier, title, metadata, and available task actions.
2. Inspect the timeline tab or default panel.
   **Expected:** Timeline rows render task-domain events in order and expose load-more or fallback behavior when applicable.
3. Switch between the available detail tabs or panels.
   **Expected:** Runs, children, dependencies, and overview content load within the same task-native route.
4. Trigger one safe navigation action, such as opening a child task, dependency, or run link.
   **Expected:** The linked route resolves correctly and keeps the operator inside the Tasks route family.

### Edge Cases & Variations

| Variation | Input / State | Expected Result |
| --- | --- | --- |
| No timeline events | seeded task with minimal history | the timeline panel shows a clean empty state |
| Load-more path | enough events for pagination | loading more does not duplicate or reorder rows |
| Draft or blocked task | non-ready task | publish/cancel/enqueue affordances match the task state instead of generic defaults |

### Related Test Cases

- `TC-FUNC-003`
- `TC-FUNC-006`
- `TC-FUNC-008`
- `TC-FUNC-009`

### Notes

- This case is part of Smoke because it proves the first task-native deep-link surface.
