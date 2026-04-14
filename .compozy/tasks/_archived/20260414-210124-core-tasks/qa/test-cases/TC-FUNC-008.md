## TC-FUNC-008: Create 65th direct child exceeds MaxDirectChildren=64

**Priority:** P1
**Type:** Functional
**Status:** Not Run
**Estimated Time:** 8 minutes
**Created:** 2026-04-14

---

### Objective
Validate that the direct child fan-out limit of MaxDirectChildren=64 is enforced. A parent task with 64 children must accept the 64th child but reject the 65th with ErrGraphLimitExceeded.

---

### Preconditions
- [ ] AGH daemon running (or test harness initialized with TaskManager and backing Store)
- [ ] ActorContext with Authority.Write=true, Authority.CreateGlobal=true
- [ ] One existing parent task with known ID

---

### Test Steps

1. **Create 64 child tasks under the parent**
   - Loop: for i in 1..64, create child task with parent_task_id=parent.ID, title="Child {i}"
   - **Expected:** All 64 children created successfully; no errors

2. **Verify parent has exactly 64 children**
   - Query TaskView for parent or count children in store
   - **Expected:** Children count == 64

3. **Attempt to create the 65th child**
   - Input: parent_task_id=parent.ID, scope=parent scope, title="Child 65 (overflow)"
   - **Expected:** Error returned; `errors.Is(err, ErrGraphLimitExceeded)` == true; error message contains "direct child count" and "64"

4. **Verify the 65th child was not persisted**
   - Query store for task with title "Child 65 (overflow)"
   - **Expected:** No such task exists

5. **Verify that creating a grandchild under one of the 64 children still works**
   - Input: parent_task_id=Child1.ID, title="Grandchild of child 1"
   - **Expected:** Task created successfully (different parent, independent limit)

---

### Edge Cases & Variations

| Variation | Input | Expected Result |
|-----------|-------|-----------------|
| Exactly 64 children | 64th child creation | Success |
| 65th child | 65th child creation | ErrGraphLimitExceeded |
| Delete a child then re-add | Remove one child, add new one | Depends on whether cancelled/deleted children count toward limit |
| Concurrent child creation | Two clients race to create the 64th and 65th | Exactly one succeeds; the other gets ErrGraphLimitExceeded |

---

### Related Test Cases
- TC-FUNC-006: Create child task under valid parent
- TC-FUNC-007: Create child at max hierarchy depth
- TC-FUNC-012: Add 33rd dependency
