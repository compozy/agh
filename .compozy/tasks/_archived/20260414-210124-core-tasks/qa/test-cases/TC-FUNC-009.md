## TC-FUNC-009: Add valid dependency edge

**Priority:** P0
**Type:** Functional
**Status:** Not Run
**Estimated Time:** 5 minutes
**Created:** 2026-04-14

---

### Objective
Validate that adding a dependency edge between two tasks with kind "blocks" persists the edge, triggers status reconciliation on the dependent task (moving it to "blocked" if the dependency is not completed), and records an audit event.

---

### Preconditions
- [ ] AGH daemon running (or test harness initialized with TaskManager and backing Store)
- [ ] Two existing tasks: Task A (status="pending") and Task B (status="pending"), both scope="global"
- [ ] ActorContext with Authority.Write=true

---

### Test Steps

1. **Add dependency: Task A depends on Task B (kind="blocks")**
   - Input:
     ```json
     {
       "task_id": "<task-A-id>",
       "depends_on_task_id": "<task-B-id>",
       "kind": "blocks"
     }
     ```
   - **Expected:** No error returned

2. **Verify the dependency edge is persisted**
   - Query dependencies for Task A
   - **Expected:** One dependency edge with DependsOnTaskID=Task B ID, Kind="blocks"

3. **Verify Task A's status was reconciled to "blocked"**
   - Read Task A from store
   - **Expected:** Task A status == "blocked" (since Task B is not completed)

4. **Verify a task.dependency_added event was recorded**
   - Query events for Task A
   - **Expected:** TaskEvent with EventType="task.dependency_added"

5. **Complete Task B, then verify Task A reconciles to "ready"**
   - Complete a run on Task B successfully
   - Read Task A from store
   - **Expected:** Task A status reconciles from "blocked" to "ready" (all dependencies satisfied)

---

### Edge Cases & Variations

| Variation | Input | Expected Result |
|-----------|-------|-----------------|
| Dependency on already-completed task | Task B status="completed" | Task A stays "pending" or "ready" (not blocked) |
| Multiple dependencies | A depends on B and C | A is "blocked" until both B and C are completed |
| Duplicate dependency edge | Add same A->B edge twice | Idempotent or rejected depending on implementation |
| Cross-scope dependency | Task A scope="global", Task B scope="workspace" | Depends on policy; verify behavior |

---

### Related Test Cases
- TC-FUNC-010: Add self-dependency
- TC-FUNC-011: Add dependency creating a cycle
- TC-FUNC-012: Add 33rd dependency
- TC-FUNC-013: Remove dependency edge
