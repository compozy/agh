## TC-FUNC-002: Tasks Inbox lane triage and approval actions

**Priority:** P0
**Type:** Functional
**Status:** Not Run
**Estimated Time:** 15 minutes
**Created:** 2026-04-17
**Last Updated:** 2026-04-17
**Module:** Tasks Inbox
**Route / Surface:** `/_app/tasks` in inbox mode
**Design Reference:** `docs/design/paper/tasks/AGH Tasks — Inbox@2x.png`
**Execution Lane:** Manual + browser regression

### Objective

Verify the inbox view exposes lane grouping, unread/archive state, approval affordances, and triage mutations without client-side grouping drift.

### Preconditions

- [ ] The app shell is reachable.
- [ ] Seed data includes inbox items across at least two lanes, including one approval or triage-ready item.
- [ ] Actor-scoped inbox state is available for the seeded workspace.

### Test Steps

1. Open Tasks and switch to inbox mode.
   **Expected:** The inbox lane tabs, totals, and grouped items appear inside the Tasks shell.
2. Toggle lane filters, unread-only state, and search.
   **Expected:** Visible groups and counts respond to the selected lane and filters without cross-mode layout regressions.
3. Perform one available inbox mutation such as approve, reject, dismiss, mark read, archive, or retry.
   **Expected:** The item updates or moves lanes according to the action, and counts refresh coherently.
4. Open one inbox item into its task detail flow.
   **Expected:** The navigation reaches the matching task detail without losing workspace or filter context.
5. Capture evidence for the lane state before and after the action.
   **Expected:** The executed mutation can be traced back to this case in screenshots or the verification report.

### Edge Cases & Variations

| Variation | Input / State | Expected Result |
| --- | --- | --- |
| No unread items | unread toggle enabled with no matches | a clear empty state is shown and lane controls remain stable |
| Archived lane | archived item exists | archive state is visible and the item is isolated from active work lanes |
| Blocking reason | blocked or failed item | the item shows the blocking reason or failure summary rather than generic status only |

### Related Test Cases

- `TC-FUNC-001`
- `TC-FUNC-007`

### Notes

- This case should run after a mutation-capable seed is available; stale counts or wrong lane movement are blocking regressions.
