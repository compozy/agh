## TC-FUNC-003: Tasks list split view browsing, filtering, and selection

**Priority:** P0
**Type:** Functional
**Status:** Not Run
**Estimated Time:** 12 minutes
**Created:** 2026-04-17
**Last Updated:** 2026-04-17
**Module:** Tasks List
**Route / Surface:** `/_app/tasks` in list mode
**Design Reference:** `docs/design/paper/tasks/AGH Tasks — List (Split View)@2x.png`
**Execution Lane:** Manual + browser regression

### Objective

Verify the Tasks sidebar entry opens the split-view list experience and that search, filters, and task selection behave coherently with the enriched task summaries.

### Preconditions

- [ ] The app shell is running.
- [ ] At least three seeded tasks exist with distinct statuses, priorities, or owners.
- [ ] One seeded task can be deep-linked into detail.

### Test Steps

1. Open the app root, resolve onboarding if needed, and click the Tasks sidebar entry.
   **Expected:** The user lands in the Tasks area and list mode is available as the primary browsing surface.
2. Verify the split layout contains a task list and a right-side preview or detail region.
   **Expected:** Enriched task cards show identifiers, status semantics, and summary metadata rather than plain titles only.
3. Apply search and one filter such as owner, status, or scope.
   **Expected:** The visible list narrows correctly and the selection state remains stable when the selected task still matches.
4. Select a task from the list.
   **Expected:** The preview panel updates to the selected task and exposes a deep-link path into task detail.
5. Follow the deep link into task detail.
   **Expected:** Navigation reaches `/_app/tasks/$id` with the selected task context intact.

### Edge Cases & Variations

| Variation | Input / State | Expected Result |
| --- | --- | --- |
| No results | unmatched search term | the list shows a no-results state without collapsing the page shell |
| Draft task | selected task is a draft | the draft state is visible and publish-related affordances remain coherent |
| Approval-backed task | selected task requires approval | approval semantics are surfaced in the card or preview, not hidden in raw metadata |

### Related Test Cases

- `TC-FUNC-005`
- `TC-FUNC-006`
- `TC-FUNC-007`

### Notes

- This case is the main browser entrypoint for Tasks and should be one of the first E2E flows implemented in task_19.
