## TC-FUNC-005: Tasks empty state templates and first-run entry paths

**Priority:** P1
**Type:** Functional
**Status:** Not Run
**Estimated Time:** 10 minutes
**Created:** 2026-04-17
**Last Updated:** 2026-04-17
**Module:** Tasks Empty State
**Route / Surface:** `/_app/tasks` with zero-task seed
**Design Reference:** `docs/design/paper/tasks/AGH Tasks — Empty State@2x.png`
**Execution Lane:** Manual + targeted browser regression

### Objective

Verify first-run Tasks entry uses the intended empty-state guidance, templates, and creation CTAs rather than a blank or degraded shell.

### Preconditions

- [ ] The app shell is running.
- [ ] The selected workspace or global scope has zero visible tasks.
- [ ] The create-modal flow is available from the empty state.

### Test Steps

1. Open Tasks with a zero-task seed.
   **Expected:** The empty-state layout renders instead of an empty list or broken split view.
2. Inspect the template options and primary CTA.
   **Expected:** The first-run guidance offers the expected template choices and a clear task-creation path.
3. Choose one template from the empty state.
   **Expected:** The create flow opens with that template preselected.
4. Return to the empty state and use the primary CTA.
   **Expected:** The CTA opens the same creation flow without breaking the shell.

### Edge Cases & Variations

| Variation | Input / State | Expected Result |
| --- | --- | --- |
| Blank template | select the blank option | the modal opens without hidden defaults that contradict the blank path |
| CLI CTA present | alternate create path is shown | the extra CTA is additive and does not replace the primary create flow |
| Workspace onboarding before zero-task state | onboarding shown first | once onboarding is resolved, the empty-state route still appears correctly |

### Related Test Cases

- `TC-FUNC-003`
- `TC-FUNC-006`

### Notes

- This case is intentionally separate from the create-modal case so task_19 can distinguish first-run UX from generic creation behavior.
