## TC-FUNC-007: Create child at max hierarchy depth (8)

**Priority:** P1
**Type:** Functional
**Status:** Not Run
**Estimated Time:** 8 minutes
**Created:** 2026-04-14

---

### Objective
Validate that the task hierarchy depth is bounded at MaxHierarchyDepth=8 levels. Creating a child that would exceed depth 8 must be rejected with ErrGraphLimitExceeded. Creating a child exactly at depth 8 must succeed.

---

### Preconditions
- [ ] AGH daemon running (or test harness initialized with TaskManager and backing Store)
- [ ] ActorContext with Authority.Write=true, Authority.CreateGlobal=true
- [ ] No pre-existing tasks

---

### Test Steps

1. **Create a chain of 8 nested tasks (depth 0 through 7)**
   - Create root task (depth 0): scope="global", title="Level 0"
   - Create child of Level 0 (depth 1): parent_task_id=Level0.ID, title="Level 1"
   - Continue chaining until Level 7 (depth 7): parent_task_id=Level6.ID, title="Level 7"
   - **Expected:** All 8 tasks created successfully; no errors

2. **Verify depth 7 task exists and is valid**
   - Read "Level 7" task from store
   - **Expected:** ParentTaskID points to "Level 6" task; all fields valid

3. **Attempt to create a child at depth 8 (Level 8)**
   - Input: parent_task_id=Level7.ID, scope="global", title="Level 8 (too deep)"
   - **Expected:** Error returned; `errors.Is(err, ErrGraphLimitExceeded)` == true; error message contains "hierarchy depth" and "8"

4. **Verify the rejected task was not persisted**
   - Query store for task with title "Level 8 (too deep)"
   - **Expected:** No such task exists

5. **Verify that a sibling at depth 7 can still be created**
   - Input: parent_task_id=Level6.ID, scope="global", title="Level 7 sibling"
   - **Expected:** Task created successfully (same depth as Level 7, not exceeding limit)

---

### Edge Cases & Variations

| Variation | Input | Expected Result |
|-----------|-------|-----------------|
| Exactly at MaxHierarchyDepth boundary | depth=7 (8th level, 0-indexed) | Depends on whether limit is inclusive; test the exact boundary |
| Deep chain with workspace scope | All tasks scope="workspace", ws="ws-1" | Same depth limit applies |
| Negative depth value | ValidateHierarchyDepth(-1) | ErrValidation (cannot be negative) |
| Zero depth (root) | ValidateHierarchyDepth(0) | Valid |

---

### Related Test Cases
- TC-FUNC-006: Create child task under valid parent
- TC-FUNC-008: Create 65th direct child
- TC-FUNC-012: Add 33rd dependency
