## TC-FUNC-003: Create task with invalid scope binding (global + workspace_id)

**Priority:** P0
**Type:** Functional
**Status:** Not Run
**Estimated Time:** 3 minutes
**Created:** 2026-04-14

---

### Objective
Validate that creating a task with scope "global" and a non-empty workspace_id is rejected with ErrInvalidScopeBinding. The scope/workspace invariant enforces that global tasks must have an empty workspace_id and workspace tasks must have a non-empty workspace_id.

---

### Preconditions
- [ ] AGH daemon running (or test harness initialized with TaskManager and backing Store)
- [ ] ActorContext with Authority.Write=true, Authority.CreateGlobal=true

---

### Test Steps

1. **Attempt to create a global task with a workspace_id**
   - Input:
     ```json
     {
       "scope": "global",
       "workspace_id": "ws-unexpected",
       "title": "Invalid binding"
     }
     ```
     ActorContext: valid with CreateGlobal authority
   - **Expected:** Error returned; `errors.Is(err, ErrInvalidScopeBinding)` == true; error message indicates workspace_id must be empty when scope is "global"

2. **Verify no task was persisted**
   - Query store for tasks
   - **Expected:** No task with title "Invalid binding" exists

3. **Verify no task event was recorded**
   - **Expected:** No task.created event in the event store

---

### Edge Cases & Variations

| Variation | Input | Expected Result |
|-----------|-------|-----------------|
| global + non-empty workspace_id | scope="global", workspace_id="ws-1" | ErrInvalidScopeBinding |
| workspace + empty workspace_id | scope="workspace", workspace_id="" | ErrInvalidScopeBinding |
| global + whitespace workspace_id | scope="global", workspace_id="   " | Depends on trim behavior; if trimmed to empty, may pass as valid global |
| Both scope and workspace_id empty | scope="", workspace_id="" | ErrValidation (scope is required) |

---

### Related Test Cases
- TC-FUNC-001: Create global task with valid fields
- TC-FUNC-002: Create workspace-scoped task
- TC-FUNC-005: Attempt to update immutable fields
