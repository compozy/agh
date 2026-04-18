## TC-FUNC-006: Tasks create modal draft and publish-aware flow

**Priority:** P0
**Type:** Functional
**Status:** Not Run
**Estimated Time:** 15 minutes
**Created:** 2026-04-17
**Last Updated:** 2026-04-17
**Module:** Tasks Create Modal
**Route / Surface:** `/_app/tasks` create modal
**Design Reference:** `docs/design/paper/tasks/AGH Tasks — Create Modal@2x.png`
**Execution Lane:** Manual + browser regression

### Objective

Verify the create modal supports template-based creation, first-class task semantics, saving a draft, and publishing into runnable state without losing route stability.

### Preconditions

- [ ] The Tasks surface is available.
- [ ] The execution seed or runtime helper can create a new task and expose it in list/detail reads.
- [ ] The workspace has permission to create and publish a task.

### Test Steps

1. Open the create modal from Tasks.
   **Expected:** The modal appears with template selection and task-semantic controls such as priority, attempts, approval, owner, and scope.
2. Select a template and fill the required fields for a draft task.
   **Expected:** Validation enables draft save only when the required inputs are valid.
3. Save the task as a draft.
   **Expected:** The modal closes, the task appears in Tasks, and draft semantics are visible in the resulting card or detail state.
4. Open the draft task and publish it through the shipped publish path.
   **Expected:** The task transitions out of draft into the correct runnable or blocked state according to its policy.
5. Verify the updated task remains accessible from list/detail flows.
   **Expected:** The published task can be inspected without stale state or duplicate entries.

### Edge Cases & Variations

| Variation | Input / State | Expected Result |
| --- | --- | --- |
| Manual approval | approval policy set to manual | publish reflects approval-aware state instead of silently forcing ready |
| Max attempts default | attempts left at default | the UI preserves the documented default and does not hide it |
| Template swap | switch templates before save | template notice text and defaults update without leaving stale values behind |

### Related Test Cases

- `TC-FUNC-003`
- `TC-FUNC-005`
- `TC-FUNC-007`

### Notes

- This case is part of Smoke and should be paired with seeded runtime helpers in task_19.
