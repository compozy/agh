## TC-INT-004: HTTP PATCH /api/v1/tasks/:id with immutable field returns 400 ErrImmutableField

**Priority:** P0
**Type:** Integration
**Status:** Not Run
**Estimated Time:** 5 minutes
**Created:** 2026-04-14

---

### Objective
Validate that HTTP PATCH `/api/tasks/:id` correctly rejects attempts to change immutable fields (created_by, origin, scope, workspace_id, parent_task_id), returns HTTP 400 with an appropriate error message, and confirms that mutable fields (title, description, metadata, network_channel, owner) are accepted.

---

### Preconditions
- [ ] AGH daemon running with all subsystems
- [ ] HTTP server listening on TCP :2123
- [ ] One task T exists (scope=global, title="Original Title", no owner)

---

### Test Steps

1. **Attempt to patch an immutable field: scope**
   - Input:
     ```http
     PATCH http://localhost:2123/api/tasks/<T.id>
     Content-Type: application/json

     {
       "scope": "workspace"
     }
     ```
   - **Expected:** HTTP 400 Bad Request
   - **Expected:** Error message contains "immutable" or the UpdateTaskRequest rejects unknown fields; the server does not silently ignore the field
   - **Note:** The UpdateTaskRequest contract only accepts title, description, metadata, network_channel, owner, clear_owner. Fields outside this set are either silently dropped by JSON binding or the `HasChanges()` check returns false, yielding a 400 "must include at least one mutable field."

2. **Attempt to patch with no mutable fields (empty patch)**
   - Input:
     ```http
     PATCH http://localhost:2123/api/tasks/<T.id>
     Content-Type: application/json

     {}
     ```
   - **Expected:** HTTP 400 Bad Request
   - **Expected:** Error message indicates "at least one mutable field" is required

3. **Attempt to patch with only immutable-named fields in body**
   - Input:
     ```http
     PATCH http://localhost:2123/api/tasks/<T.id>
     Content-Type: application/json

     {
       "created_by": {"kind": "automation", "ref": "attacker"},
       "origin": {"kind": "network", "ref": "spoofed"}
     }
     ```
   - **Expected:** HTTP 400 Bad Request
   - **Expected:** Since created_by and origin are not in UpdateTaskRequest, `HasChanges()` returns false, yielding "must include at least one mutable field"
   - **Expected:** The task's actual `created_by` and `origin` remain unchanged (verify with GET)

4. **Verify task is unchanged after rejected patches**
   - Input: `GET http://localhost:2123/api/tasks/<T.id>`
   - **Expected:** HTTP 200
   - **Expected:** `task.task.title` still equals "Original Title"
   - **Expected:** `task.task.scope` still equals "global"
   - **Expected:** `task.task.created_by` still equals the original server-derived identity
   - **Expected:** `task.task.origin` still equals the original server-derived origin

5. **Successfully patch a mutable field: title**
   - Input:
     ```http
     PATCH http://localhost:2123/api/tasks/<T.id>
     Content-Type: application/json

     {
       "title": "Updated Title"
     }
     ```
   - **Expected:** HTTP 200
   - **Expected:** `task.title` equals "Updated Title"
   - **Expected:** `task.updated_at` is later than previous `updated_at`
   - **Expected:** `task.scope`, `task.created_by`, `task.origin` remain unchanged

6. **Successfully patch mutable field: description**
   - Input:
     ```http
     PATCH http://localhost:2123/api/tasks/<T.id>
     Content-Type: application/json

     {
       "description": "A new description"
     }
     ```
   - **Expected:** HTTP 200
   - **Expected:** `task.description` equals "A new description"

7. **Successfully set and then clear owner**
   - Input (set owner):
     ```http
     PATCH http://localhost:2123/api/tasks/<T.id>
     Content-Type: application/json

     {
       "owner": {"kind": "human", "ref": "bob"}
     }
     ```
   - **Expected:** HTTP 200, `task.owner.kind` = "human", `task.owner.ref` = "bob"
   - Input (clear owner):
     ```http
     PATCH http://localhost:2123/api/tasks/<T.id>
     Content-Type: application/json

     {
       "clear_owner": true
     }
     ```
   - **Expected:** HTTP 200, `task.owner` is null

8. **Reject combined owner + clear_owner**
   - Input:
     ```http
     PATCH http://localhost:2123/api/tasks/<T.id>
     Content-Type: application/json

     {
       "owner": {"kind": "human", "ref": "charlie"},
       "clear_owner": true
     }
     ```
   - **Expected:** HTTP 400 (cannot set both owner and clear_owner)

9. **Reject blank title**
   - Input:
     ```http
     PATCH http://localhost:2123/api/tasks/<T.id>
     Content-Type: application/json

     {
       "title": ""
     }
     ```
   - **Expected:** HTTP 400 (title is required when provided)

---

### Data Validation
| Field | Source Value | Expected Value | Status |
|-------|-------------|----------------|--------|
| Immutable scope change | PATCH {scope: "workspace"} | 400 rejected | [ ] |
| Empty patch | PATCH {} | 400 rejected | [ ] |
| Immutable created_by/origin | PATCH with those fields | 400 rejected | [ ] |
| Mutable title | PATCH {title: "Updated Title"} | 200, title updated | [ ] |
| Mutable description | PATCH {description: "..."} | 200, description updated | [ ] |
| Set owner | PATCH {owner: {...}} | 200, owner set | [ ] |
| Clear owner | PATCH {clear_owner: true} | 200, owner null | [ ] |
| Combined owner+clear_owner | PATCH with both | 400 rejected | [ ] |
| Blank title | PATCH {title: ""} | 400 rejected | [ ] |

---

### Error Scenarios
- [ ] PATCH on non-existent task ID returns 404 (ErrTaskNotFound)
- [ ] PATCH with invalid JSON body returns 400 (decode error)
- [ ] PATCH with metadata exceeding 16 KiB returns 413 (ErrPayloadTooLarge)

---

### Related Test Cases
- TC-INT-001: Creates the task under test
- TC-INT-003: Verifies the full detail payload after updates
- TC-INT-005: UDS endpoint parity for PATCH behavior
