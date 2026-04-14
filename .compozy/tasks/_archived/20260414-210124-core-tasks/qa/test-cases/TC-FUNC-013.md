## TC-FUNC-013: Remove dependency edge triggers status reconciliation

**Priority:** P0
**Type:** Functional
**Status:** Not Run
**Estimated Time:** 5 minutes
**Created:** 2026-04-14

---

### Objective
Validate that removing a dependency edge from a task persists the removal, records an audit event, and triggers status reconciliation. Specifically, when the last blocking dependency is removed from a "blocked" task, the task should reconcile to "ready".

---

### Preconditions
- [ ] AGH daemon running (or test harness initialized with TaskManager and backing Store)
- [ ] Task A (status="blocked") with exactly one dependency: A depends on Task B (status="pending")
- [ ] ActorContext with Authority.Write=true

---

### Test Steps

1. **Verify initial state**
   - Read Task A from store
   - **Expected:** Status == "blocked"; one dependency edge A->B exists

2. **Remove the dependency: A no longer depends on B**
   - Call RemoveDependency(ctx, taskA.ID, taskB.ID, actor)
   - **Expected:** No error returned

3. **Verify the dependency edge is removed**
   - Query dependencies for Task A
   - **Expected:** No dependency edges exist for Task A

4. **Verify Task A status reconciled from "blocked" to "ready"**
   - Read Task A from store
   - **Expected:** Status == "ready" (no remaining blocking dependencies, assuming no other blockers)

5. **Verify a task.dependency_removed event was recorded**
   - Query events for Task A
   - **Expected:** TaskEvent with EventType="task.dependency_removed"

6. **Attempt to remove a non-existent dependency**
   - Call RemoveDependency(ctx, taskA.ID, "nonexistent-task", actor)
   - **Expected:** Error returned (ErrTaskDependencyNotFound or similar)

---

### Edge Cases & Variations

| Variation | Input | Expected Result |
|-----------|-------|-----------------|
| Remove one of multiple deps | A depends on B and C; remove A->B | A stays "blocked" (still depends on C) |
| Remove last dep | A depends only on B; remove A->B | A reconciles from "blocked" to "ready" |
| Remove dep from non-blocked task | A is "pending" with dependency on completed B; remove A->B | A remains in current reconciled state |
| Remove dependency that was already completed | A depends on completed B; remove edge | No status change (was already unblocked by B's completion) |

---

### Related Test Cases
- TC-FUNC-009: Add valid dependency edge
- TC-FUNC-012: Add 33rd dependency
- TC-FUNC-014: Enqueue run on ready task
