## TC-FUNC-002: Create workspace-scoped task

**Priority:** P0
**Type:** Functional
**Status:** Not Run
**Estimated Time:** 5 minutes
**Created:** 2026-04-14

---

### Objective
Validate that creating a task with scope "workspace" requires a non-empty workspace_id, produces a persisted task bound to that workspace, and that omitting workspace_id with scope "workspace" is rejected with ErrInvalidScopeBinding.

---

### Preconditions
- [ ] AGH daemon running (or test harness initialized with TaskManager and backing Store)
- [ ] ActorContext with Authority.Write=true, Authority.CreateWorkspace=true
- [ ] A valid workspace identifier "ws-abc123" available

---

### Test Steps

1. **Create a workspace-scoped task with valid workspace_id**
   - Input:
     ```json
     {
       "scope": "workspace",
       "workspace_id": "ws-abc123",
       "title": "Implement feature X"
     }
     ```
     ActorContext: Actor={Kind:"human", Ref:"user-1"}, Origin={Kind:"web", Ref:"browser-session-1"}, Authority={Read:true, Write:true, CreateWorkspace:true}
   - **Expected:** No error returned

2. **Inspect the returned Task record**
   - **Expected:**
     - `task.Scope` == "workspace"
     - `task.WorkspaceID` == "ws-abc123"
     - `task.Status` == "pending"
     - `task.Title` == "Implement feature X"
     - All server-derived fields (ID, CreatedAt, UpdatedAt, CreatedBy, Origin) populated correctly

3. **Attempt to create workspace-scoped task without workspace_id**
   - Input:
     ```json
     {
       "scope": "workspace",
       "workspace_id": "",
       "title": "Missing workspace"
     }
     ```
   - **Expected:** Error wrapping ErrInvalidScopeBinding; message indicates workspace_id is required when scope is "workspace"

4. **Attempt to create workspace-scoped task with whitespace-only workspace_id**
   - Input: scope="workspace", workspace_id="   ", title="Whitespace workspace"
   - **Expected:** Error wrapping ErrInvalidScopeBinding

---

### Edge Cases & Variations

| Variation | Input | Expected Result |
|-----------|-------|-----------------|
| Valid workspace binding | scope="workspace", workspace_id="ws-1" | Task persisted with workspace binding |
| Empty workspace_id | scope="workspace", workspace_id="" | ErrInvalidScopeBinding |
| Whitespace workspace_id | scope="workspace", workspace_id="  " | ErrInvalidScopeBinding |
| Missing CreateWorkspace authority | Authority.CreateWorkspace=false | ErrPermissionDenied |
| Very long workspace_id | scope="workspace", workspace_id=string(500 chars) | Task created (no length limit on workspace_id itself) |

---

### Related Test Cases
- TC-FUNC-001: Create global task with valid fields
- TC-FUNC-003: Create task with invalid scope binding
