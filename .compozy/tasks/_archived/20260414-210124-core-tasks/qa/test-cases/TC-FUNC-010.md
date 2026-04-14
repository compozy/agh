## TC-FUNC-010: Add self-dependency rejected

**Priority:** P1
**Type:** Functional
**Status:** Not Run
**Estimated Time:** 3 minutes
**Created:** 2026-04-14

---

### Objective
Validate that adding a dependency where task_id == depends_on_task_id is rejected at the validation layer with ErrValidation. A task cannot depend on itself.

---

### Preconditions
- [ ] AGH daemon running (or test harness initialized with TaskManager and backing Store)
- [ ] One existing task with known ID
- [ ] ActorContext with Authority.Write=true

---

### Test Steps

1. **Attempt to add a self-dependency**
   - Input:
     ```json
     {
       "task_id": "<task-id>",
       "depends_on_task_id": "<task-id>",
       "kind": "blocks"
     }
     ```
   - **Expected:** Error returned; `errors.Is(err, ErrValidation)` == true; error message contains "cannot depend on itself"

2. **Verify no dependency edge was persisted**
   - Query dependencies for the task
   - **Expected:** No dependency edges exist

3. **Verify no task.dependency_added event was recorded**
   - **Expected:** No events of type "task.dependency_added" for this task

4. **Verify the task status is unchanged**
   - Read the task from store
   - **Expected:** Status remains at its pre-test value (not "blocked")

---

### Edge Cases & Variations

| Variation | Input | Expected Result |
|-----------|-------|-----------------|
| Self-dependency with whitespace | task_id=" task-1 ", depends_on=" task-1 " | Rejected after TrimSpace normalization |
| Self-dependency via AddDependency.Validate | Direct validation call | ErrValidation |
| Self-dependency via TaskDependency.Validate | Direct struct validation | ErrValidation |

---

### Related Test Cases
- TC-FUNC-009: Add valid dependency edge
- TC-FUNC-011: Add dependency creating a cycle
