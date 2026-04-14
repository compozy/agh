## TC-FUNC-001: Create global task with valid fields

**Priority:** P0
**Type:** Functional
**Status:** Not Run
**Estimated Time:** 5 minutes
**Created:** 2026-04-14

---

### Objective
Validate that creating a task with scope "global", a non-empty title, valid actor identity, and valid origin produces a persisted task record with server-derived ID, status "pending", correct timestamps, and all supplied fields stored verbatim.

---

### Preconditions
- [ ] AGH daemon running (or test harness initialized with TaskManager and backing Store)
- [ ] ActorContext with Authority.Write=true, Authority.CreateGlobal=true
- [ ] No pre-existing tasks in the store

---

### Test Steps

1. **Create a global task via TaskManager.CreateTask**
   - Input:
     ```json
     {
       "scope": "global",
       "title": "Bootstrap infrastructure",
       "description": "Set up core services",
       "metadata": {"priority": "high"}
     }
     ```
     ActorContext: Actor={Kind:"human", Ref:"user-1"}, Origin={Kind:"cli", Ref:"terminal-1"}, Authority={Read:true, Write:true, CreateGlobal:true}
   - **Expected:** No error returned

2. **Inspect the returned Task record**
   - **Expected:**
     - `task.ID` is non-empty and server-generated (prefixed with task domain prefix)
     - `task.Scope` == "global"
     - `task.WorkspaceID` == "" (empty for global)
     - `task.Title` == "Bootstrap infrastructure"
     - `task.Description` == "Set up core services"
     - `task.Status` == "pending"
     - `task.CreatedBy` == {Kind:"human", Ref:"user-1"}
     - `task.Origin` == {Kind:"cli", Ref:"terminal-1"}
     - `task.CreatedAt` is non-zero and close to now
     - `task.UpdatedAt` == `task.CreatedAt`
     - `task.Metadata` contains `{"priority": "high"}`
     - `task.ParentTaskID` == ""
     - `task.Owner` == nil

3. **Read the task back from the store by ID**
   - **Expected:** Returned task matches all fields from step 2 exactly

4. **Verify a task.created event was recorded**
   - **Expected:** TaskEvent exists with EventType="task.created", TaskID=task.ID, Actor and Origin matching the ActorContext

---

### Edge Cases & Variations

| Variation | Input | Expected Result |
|-----------|-------|-----------------|
| Minimal valid fields | scope="global", title="x" (no description, no metadata, no owner) | Task created with empty description, nil metadata, nil owner |
| With optional owner | scope="global", title="x", owner={kind:"human", ref:"owner-1"} | Task created with owner set |
| With optional identifier | scope="global", title="x", identifier="INFRA-001" | Task created with identifier stored |
| With network_channel | scope="global", title="x", network_channel="chan-1" | Task created with network_channel set (if channel validator passes) |
| Empty title | scope="global", title="" | ErrValidation returned |
| Missing scope | scope="", title="x" | ErrValidation returned |
| Invalid scope value | scope="unknown", title="x" | ErrValidation returned |

---

### Related Test Cases
- TC-FUNC-002: Create workspace-scoped task
- TC-FUNC-003: Create task with invalid scope binding
- TC-FUNC-004: Update mutable fields
