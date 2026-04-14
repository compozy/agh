## TC-FUNC-005: Attempt to update immutable fields (scope, workspace_id, parent_task_id, created_by, origin)

**Priority:** P0
**Type:** Functional
**Status:** Not Run
**Estimated Time:** 8 minutes
**Created:** 2026-04-14

---

### Objective
Validate that each immutable task field (scope, workspace_id, parent_task_id, created_by_kind, created_by_ref, origin_kind, origin_ref) cannot be changed after task creation. Each attempt must return ErrImmutableField with the field name in the error message.

---

### Preconditions
- [ ] AGH daemon running (or test harness initialized with TaskManager and backing Store)
- [ ] One existing global task (scope="global", workspace_id="") created by actor {Kind:"human", Ref:"creator-1"} via origin {Kind:"cli", Ref:"term-1"}, with no parent
- [ ] Direct access to ValidateImmutableTaskFields or equivalent update path

---

### Test Steps

1. **Attempt to change scope from "global" to "workspace"**
   - Construct a modified task record with Scope="workspace"
   - Call ValidateImmutableTaskFields(original, modified)
   - **Expected:** Error returned; `errors.Is(err, ErrImmutableField)` == true; error message contains "scope"

2. **Attempt to change workspace_id from "" to "ws-1"**
   - Construct a modified task record with WorkspaceID="ws-1"
   - **Expected:** Error returned; `errors.Is(err, ErrImmutableField)` == true; error message contains "workspace_id"

3. **Attempt to change parent_task_id from "" to "task-parent"**
   - Construct a modified task record with ParentTaskID="task-parent"
   - **Expected:** Error returned; `errors.Is(err, ErrImmutableField)` == true; error message contains "parent_task_id"

4. **Attempt to change created_by kind from "human" to "agent_session"**
   - Construct a modified task record with CreatedBy.Kind="agent_session"
   - **Expected:** Error returned; `errors.Is(err, ErrImmutableField)` == true; error message contains "created_by"

5. **Attempt to change created_by ref from "creator-1" to "creator-2"**
   - Construct a modified task record with CreatedBy.Ref="creator-2"
   - **Expected:** Error returned; `errors.Is(err, ErrImmutableField)` == true; error message contains "created_by"

6. **Attempt to change origin kind from "cli" to "web"**
   - Construct a modified task record with Origin.Kind="web"
   - **Expected:** Error returned; `errors.Is(err, ErrImmutableField)` == true; error message contains "origin"

7. **Attempt to change origin ref from "term-1" to "browser-1"**
   - Construct a modified task record with Origin.Ref="browser-1"
   - **Expected:** Error returned; `errors.Is(err, ErrImmutableField)` == true; error message contains "origin"

8. **Verify no changes were persisted for any failed attempt**
   - Read the task back from the store
   - **Expected:** All fields match the original values exactly

---

### Edge Cases & Variations

| Variation | Input | Expected Result |
|-----------|-------|-----------------|
| Same immutable value (no-op) | scope remains "global" | No error (values unchanged) |
| Case-sensitive scope change | scope="Global" vs "global" | Depends on normalization; after Normalize() they match, so no error |
| Whitespace-padded ref | created_by.ref=" creator-1 " vs "creator-1" | Depends on TrimSpace; if trimmed values match, no error |
| Multiple immutable fields changed | scope + workspace_id both changed | First immutable violation detected is reported |

---

### Related Test Cases
- TC-FUNC-004: Update mutable fields
- TC-FUNC-001: Create global task with valid fields
