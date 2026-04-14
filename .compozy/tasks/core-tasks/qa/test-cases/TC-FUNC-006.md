## TC-FUNC-006: Create child task under valid parent

**Priority:** P0
**Type:** Functional
**Status:** Not Run
**Estimated Time:** 5 minutes
**Created:** 2026-04-14

---

### Objective
Validate that creating a task with a valid parent_task_id sets the parent linkage correctly, that the child inherits the parent's scope and workspace binding, and that depth is tracked and validated within the MaxHierarchyDepth limit of 8 levels.

---

### Preconditions
- [ ] AGH daemon running (or test harness initialized with TaskManager and backing Store)
- [ ] One existing parent task with known ID (e.g., scope="workspace", workspace_id="ws-1")
- [ ] ActorContext with Authority.Write=true, Authority.CreateWorkspace=true

---

### Test Steps

1. **Create a child task under a valid parent**
   - Input:
     ```json
     {
       "scope": "workspace",
       "workspace_id": "ws-1",
       "parent_task_id": "<parent-task-id>",
       "title": "Child task"
     }
     ```
   - **Expected:** No error returned

2. **Inspect the returned child Task record**
   - **Expected:**
     - `task.ParentTaskID` == parent task ID
     - `task.Scope` == parent's scope ("workspace")
     - `task.WorkspaceID` == parent's workspace_id ("ws-1")
     - `task.Status` == "pending"
     - Server-derived ID is different from parent ID

3. **Verify a task.child_created event was recorded on the parent**
   - **Expected:** TaskEvent with EventType="task.child_created" and TaskID=parent task ID

4. **Create a grandchild task (depth=2)**
   - Input: parent_task_id=child task ID, scope="workspace", workspace_id="ws-1", title="Grandchild"
   - **Expected:** Task created successfully at depth 2

5. **Verify the parent task's children list includes the child**
   - Query TaskView for parent
   - **Expected:** Children array contains the child task summary

---

### Edge Cases & Variations

| Variation | Input | Expected Result |
|-----------|-------|-----------------|
| Child of global parent | parent scope="global", child scope="global" | Child created with no workspace_id |
| Child scope mismatch | parent scope="workspace", child scope="global" | Rejected; child must match parent scope |
| Child workspace mismatch | parent ws_id="ws-1", child ws_id="ws-2" | Rejected; child must match parent workspace |
| Nonexistent parent | parent_task_id="nonexistent" | ErrTaskNotFound |
| Self-referencing parent | parent_task_id == own ID (client-supplied) | ErrValidation (cannot equal own ID) |

---

### Related Test Cases
- TC-FUNC-007: Create child at max hierarchy depth
- TC-FUNC-008: Create 65th direct child
- TC-FUNC-001: Create global task with valid fields
