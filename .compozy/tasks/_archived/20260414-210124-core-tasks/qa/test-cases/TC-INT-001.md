## TC-INT-001: HTTP POST /api/v1/tasks with valid JSON returns 201 with server-derived identity

**Priority:** P0
**Type:** Integration
**Status:** Not Run
**Estimated Time:** 5 minutes
**Created:** 2026-04-14

---

### Objective
Validate that an HTTP POST to `/api/tasks` with a well-formed CreateTaskRequest body returns HTTP 201 and a TaskPayload whose server-derived fields (id, created_by, origin, status, timestamps) are correctly populated regardless of caller-supplied values.

---

### Preconditions
- [ ] AGH daemon running with all subsystems (task manager, store, session executor)
- [ ] HTTP server listening on TCP :2123
- [ ] No pre-existing tasks in the store (clean state)

---

### Test Steps

1. **Create a global-scope task with minimal required fields**
   - Input:
     ```http
     POST http://localhost:2123/api/tasks
     Content-Type: application/json

     {
       "scope": "global",
       "title": "TC-INT-001 global task"
     }
     ```
   - **Expected:** HTTP 201 Created
   - **Expected:** Response body contains `{"task": {...}}` with a TaskPayload

2. **Verify server-derived identity fields on the response**
   - **Expected:** `task.id` is a non-empty string (server-generated UUID)
   - **Expected:** `task.status` equals `"pending"` (initial lifecycle state)
   - **Expected:** `task.created_by.kind` equals `"human"` (default actor for HTTP ingress)
   - **Expected:** `task.created_by.ref` equals `"local-user"` (default actor ref)
   - **Expected:** `task.origin.kind` equals `"http"` (HTTP transport origin)
   - **Expected:** `task.origin.ref` is a non-empty string containing `"tasks.create"`
   - **Expected:** `task.created_at` is a valid RFC3339 timestamp within the last 5 seconds
   - **Expected:** `task.updated_at` is a valid RFC3339 timestamp equal to or after `created_at`
   - **Expected:** `task.closed_at` is null (task is not closed)

3. **Verify echo of caller-supplied fields**
   - **Expected:** `task.scope` equals `"global"`
   - **Expected:** `task.title` equals `"TC-INT-001 global task"`
   - **Expected:** `task.workspace_id` is empty (global scope has no workspace binding)
   - **Expected:** `task.parent_task_id` is empty (no parent specified)
   - **Expected:** `task.description` is empty (not supplied)
   - **Expected:** `task.owner` is null (not supplied)
   - **Expected:** `task.metadata` is null or empty (not supplied)

4. **Create a workspace-scope task with all optional fields**
   - Input:
     ```http
     POST http://localhost:2123/api/tasks
     Content-Type: application/json

     {
       "scope": "workspace",
       "workspace": "<valid-workspace-ref>",
       "title": "TC-INT-001 workspace task",
       "description": "Full task with all optional fields",
       "identifier": "test-001",
       "owner": {"kind": "human", "ref": "alice"},
       "metadata": {"priority": "high", "labels": ["qa", "integration"]}
     }
     ```
   - **Expected:** HTTP 201 Created
   - **Expected:** `task.scope` equals `"workspace"`
   - **Expected:** `task.workspace_id` is a resolved non-empty workspace ID
   - **Expected:** `task.identifier` equals `"test-001"`
   - **Expected:** `task.description` equals `"Full task with all optional fields"`
   - **Expected:** `task.owner.kind` equals `"human"` and `task.owner.ref` equals `"alice"`
   - **Expected:** `task.metadata` is the echoed JSON object

5. **Verify the created task persists via GET**
   - Input: `GET http://localhost:2123/api/tasks/<id-from-step-1>`
   - **Expected:** HTTP 200 with TaskDetailPayload matching the created task

---

### Data Validation
| Field | Source Value | Expected Value | Status |
|-------|-------------|----------------|--------|
| task.id | (server-generated) | Non-empty UUID string | [ ] |
| task.status | (server-derived) | "pending" | [ ] |
| task.created_by.kind | (server-derived) | "human" | [ ] |
| task.created_by.ref | (server-derived) | "local-user" | [ ] |
| task.origin.kind | (server-derived) | "http" | [ ] |
| task.origin.ref | (server-derived) | Contains "tasks.create" | [ ] |
| task.scope | "global" | "global" | [ ] |
| task.title | "TC-INT-001 global task" | "TC-INT-001 global task" | [ ] |
| task.created_at | (server-derived) | Valid RFC3339 within 5s | [ ] |
| task.updated_at | (server-derived) | >= created_at | [ ] |
| task.closed_at | (server-derived) | null | [ ] |

---

### Error Scenarios
- [ ] POST with empty body returns 400 (missing required fields: scope, title)
- [ ] POST with `scope: "workspace"` but no `workspace` field returns 400 (ErrInvalidScopeBinding)
- [ ] POST with `scope: "global"` and a `workspace` value returns 400 (ErrInvalidScopeBinding)
- [ ] POST with `scope: "invalid_value"` returns 400 (ErrValidation)
- [ ] POST with title as empty string returns 400 (ErrValidation)
- [ ] POST with metadata exceeding 16 KiB returns 413 (ErrPayloadTooLarge)

---

### Related Test Cases
- TC-INT-003: GET /api/tasks/:id validates the full TaskDetailPayload shape
- TC-INT-005: UDS endpoint parity confirms the same 201 behavior on UDS transport
- TC-INT-006: CLI task create exercises the same endpoint via daemon API
