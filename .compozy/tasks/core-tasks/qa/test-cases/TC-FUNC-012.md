## TC-FUNC-012: Add 33rd dependency exceeds MaxDependencyCount=32

**Priority:** P1
**Type:** Functional
**Status:** Not Run
**Estimated Time:** 8 minutes
**Created:** 2026-04-14

---

### Objective
Validate that a task cannot have more than MaxDependencyCount=32 dependency edges. Adding the 33rd dependency must be rejected with ErrGraphLimitExceeded.

---

### Preconditions
- [ ] AGH daemon running (or test harness initialized with TaskManager and backing Store)
- [ ] 33 existing tasks: one dependent task (Task A) and 33 potential dependency targets (Task D1..D33)
- [ ] ActorContext with Authority.Write=true

---

### Test Steps

1. **Add 32 dependencies to Task A**
   - Loop: for i in 1..32, add dependency A->Di (kind="blocks")
   - **Expected:** All 32 dependencies added successfully; no errors

2. **Verify Task A has exactly 32 dependencies**
   - Query dependencies for Task A
   - **Expected:** 32 dependency edges returned

3. **Verify Task A is "blocked"**
   - Read Task A from store
   - **Expected:** Task A status == "blocked" (32 pending dependencies)

4. **Attempt to add the 33rd dependency (A->D33)**
   - Input:
     ```json
     {
       "task_id": "<task-A-id>",
       "depends_on_task_id": "<task-D33-id>",
       "kind": "blocks"
     }
     ```
   - **Expected:** Error returned; `errors.Is(err, ErrGraphLimitExceeded)` == true; error message contains "dependency count" and "32"

5. **Verify the 33rd edge was not persisted**
   - Query dependencies for Task A
   - **Expected:** Still exactly 32 edges

6. **Remove one dependency, then add a new one**
   - Remove A->D1 dependency
   - Add A->D33 dependency
   - **Expected:** Both operations succeed; Task A now has 32 dependencies (D2..D33)

---

### Edge Cases & Variations

| Variation | Input | Expected Result |
|-----------|-------|-----------------|
| Exactly 32 dependencies | 32nd addition | Success |
| 33rd dependency | 33rd addition | ErrGraphLimitExceeded |
| Remove then re-add | Remove one, add new | Success (back to 32) |
| ValidateDependencyCount(33) | Direct validation call | ErrGraphLimitExceeded |
| ValidateDependencyCount(32) | Direct validation call | No error |

---

### Related Test Cases
- TC-FUNC-009: Add valid dependency edge
- TC-FUNC-013: Remove dependency edge
- TC-FUNC-007: Create child at max hierarchy depth
- TC-FUNC-008: Create 65th direct child
