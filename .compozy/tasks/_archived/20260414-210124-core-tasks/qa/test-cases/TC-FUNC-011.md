## TC-FUNC-011: Add dependency creating a cycle (A->B->C->A) rejected

**Priority:** P0
**Type:** Functional
**Status:** Not Run
**Estimated Time:** 8 minutes
**Created:** 2026-04-14

---

### Objective
Validate that adding a dependency edge that would create a cycle in the dependency graph is detected and rejected with ErrCycleDetected. This covers both direct two-node cycles (A->B->A) and transitive multi-node cycles (A->B->C->A).

---

### Preconditions
- [ ] AGH daemon running (or test harness initialized with TaskManager and backing Store)
- [ ] Three existing tasks: Task A, Task B, Task C (all scope="global", status="pending")
- [ ] ActorContext with Authority.Write=true

---

### Test Steps

1. **Set up the chain: A depends on B, B depends on C**
   - Add dependency: A->B (kind="blocks") -- **Expected:** Success
   - Add dependency: B->C (kind="blocks") -- **Expected:** Success

2. **Verify the chain is established**
   - Query dependencies for A: **Expected:** A depends on B
   - Query dependencies for B: **Expected:** B depends on C

3. **Attempt to close the cycle: C depends on A**
   - Input:
     ```json
     {
       "task_id": "<task-C-id>",
       "depends_on_task_id": "<task-A-id>",
       "kind": "blocks"
     }
     ```
   - **Expected:** Error returned; `errors.Is(err, ErrCycleDetected)` == true

4. **Verify no C->A edge was persisted**
   - Query dependencies for Task C
   - **Expected:** No dependency edges exist for Task C

5. **Verify Task C status is unchanged**
   - **Expected:** Task C status remains "pending" (not "blocked")

6. **Test direct two-node cycle: B depends on A (while A depends on B)**
   - Input: task_id=B, depends_on_task_id=A, kind="blocks"
   - **Expected:** Error returned; `errors.Is(err, ErrCycleDetected)` == true

---

### Edge Cases & Variations

| Variation | Input | Expected Result |
|-----------|-------|-----------------|
| Direct 2-node cycle (A->B, B->A) | B depends on A after A depends on B | ErrCycleDetected |
| 3-node cycle (A->B->C->A) | C depends on A after chain | ErrCycleDetected |
| Long chain (A->B->C->D->E->A) | E depends on A after 4-hop chain | ErrCycleDetected |
| Diamond (no cycle): A->B, A->C, B->D, C->D | D->A would be a cycle; D->B is not | D->A: ErrCycleDetected; legitimate diamond: no error |
| Disconnected graphs | Add dependency in separate subgraph | No cycle; succeeds |

---

### Related Test Cases
- TC-FUNC-009: Add valid dependency edge
- TC-FUNC-010: Add self-dependency
- TC-FUNC-012: Add 33rd dependency
