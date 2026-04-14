## TC-INT-005: UDS endpoint parity -- all 18 task endpoints return identical results as HTTP

**Priority:** P0
**Type:** Integration
**Status:** Not Run
**Estimated Time:** 15 minutes
**Created:** 2026-04-14

---

### Objective
Validate that all 18 task-related endpoints registered on the UDS transport (`/tmp/.agh/daemon.sock`) produce identical response shapes, status codes, and data as the HTTP transport (TCP :2123), differing only in the `origin.kind` field (`"uds"` vs `"http"`).

---

### Preconditions
- [ ] AGH daemon running with all subsystems
- [ ] HTTP server listening on TCP :2123
- [ ] UDS server listening on `/tmp/.agh/daemon.sock`
- [ ] At least one workspace registered (for workspace-scoped test data)
- [ ] Clean task store (no pre-existing tasks)

---

### Test Steps

Each step issues the same request to both transports and compares the responses.

1. **POST /api/tasks (CreateTask)**
   - HTTP: `POST http://localhost:2123/api/tasks` with `{"scope":"global","title":"Parity Test"}`
   - UDS: `curl --unix-socket /tmp/.agh/daemon.sock http://localhost/api/tasks` with same body
   - **Expected:** Both return 201
   - **Expected:** Both return TaskPayload with identical field shapes
   - **Expected:** HTTP task has `origin.kind = "http"`, UDS task has `origin.kind = "uds"`
   - Record both task IDs as HTTP_TASK_ID and UDS_TASK_ID

2. **GET /api/tasks (ListTasks)**
   - Both transports: `GET /api/tasks`
   - **Expected:** Both return 200 with `{"tasks": [...]}`
   - **Expected:** Both contain the 2 tasks just created
   - **Expected:** Array element shapes are identical TaskSummaryPayload

3. **GET /api/tasks/:id (GetTask)**
   - HTTP: `GET /api/tasks/<HTTP_TASK_ID>` via HTTP
   - UDS: `GET /api/tasks/<UDS_TASK_ID>` via UDS
   - **Expected:** Both return 200 with TaskDetailPayload
   - **Expected:** Detail payload includes task, children (empty), dependencies (empty), runs (empty), events (non-empty: task_created)

4. **PATCH /api/tasks/:id (UpdateTask)**
   - Both transports: `PATCH /api/tasks/<respective_id>` with `{"title":"Updated Parity"}`
   - **Expected:** Both return 200 with updated TaskPayload
   - **Expected:** `task.title` equals "Updated Parity" on both

5. **POST /api/tasks/:id/cancel (CancelTask)**
   - Create two fresh tasks (one per transport) for this step
   - Both: `POST /api/tasks/<id>/cancel` with `{"reason":"parity test"}`
   - **Expected:** Both return 200 with TaskPayload where `status = "cancelled"`

6. **POST /api/tasks/:id/children (CreateChildTask)**
   - Create parent tasks on each transport
   - Both: `POST /api/tasks/<parent_id>/children` with `{"scope":"global","title":"Child Parity"}`
   - **Expected:** Both return 201 with TaskPayload where `parent_task_id` matches

7. **POST /api/tasks/:id/dependencies (AddTaskDependency)**
   - Create dependency target tasks on each transport
   - Both: `POST /api/tasks/<id>/dependencies` with `{"depends_on_task_id":"<dep_id>"}`
   - **Expected:** Both return 200 with TaskDetailPayload including the dependency

8. **DELETE /api/tasks/:id/dependencies/:depends_on_id (RemoveTaskDependency)**
   - Both: `DELETE /api/tasks/<id>/dependencies/<dep_id>`
   - **Expected:** Both return 200 with TaskDetailPayload, dependency removed

9. **POST /api/tasks/:id/runs (EnqueueTaskRun)**
   - Create ready tasks on each transport
   - Both: `POST /api/tasks/<id>/runs`
   - **Expected:** Both return 201 with TaskRunPayload, `status = "queued"`

10. **GET /api/tasks/:id/runs (ListTaskRuns)**
    - Both: `GET /api/tasks/<id>/runs`
    - **Expected:** Both return 200 with `{"runs": [...]}`

11. **POST /api/task-runs/:id/claim (ClaimTaskRun)**
    - Both: `POST /api/task-runs/<run_id>/claim`
    - **Expected:** Both return 200 with TaskRunPayload, `status = "claimed"`

12. **POST /api/task-runs/:id/start (StartTaskRun)**
    - Both: `POST /api/task-runs/<run_id>/start`
    - **Expected:** Both return 200, `status = "starting"`

13. **POST /api/task-runs/:id/attach-session (AttachTaskRunSession)**
    - Both: `POST /api/task-runs/<run_id>/attach-session` with `{"session_id":"<valid_session>"}`
    - **Expected:** Both return 200, `session_id` populated

14. **POST /api/task-runs/:id/complete (CompleteTaskRun)**
    - Progress runs to "running" on each transport, then:
    - Both: `POST /api/task-runs/<run_id>/complete` with `{"result":{"ok":true}}`
    - **Expected:** Both return 200, `status = "completed"`

15. **POST /api/task-runs/:id/fail (FailTaskRun)**
    - Create and progress separate runs to "running"
    - Both: `POST /api/task-runs/<run_id>/fail` with `{"error":"test failure"}`
    - **Expected:** Both return 200, `status = "failed"`

16. **POST /api/task-runs/:id/cancel (CancelTaskRun)**
    - Create and progress separate runs to "claimed"
    - Both: `POST /api/task-runs/<run_id>/cancel` with `{"reason":"no longer needed"}`
    - **Expected:** Both return 200, `status = "cancelled"`

---

### Data Validation
| Field | HTTP Value | UDS Value | Status |
|-------|-----------|-----------|--------|
| CreateTask status code | 201 | 201 | [ ] |
| CreateTask response shape | TaskPayload | TaskPayload | [ ] |
| origin.kind on create | "http" | "uds" | [ ] |
| ListTasks status code | 200 | 200 | [ ] |
| GetTask status code | 200 | 200 | [ ] |
| GetTask detail shape | TaskDetailPayload | TaskDetailPayload | [ ] |
| UpdateTask status code | 200 | 200 | [ ] |
| CancelTask status code | 200 | 200 | [ ] |
| EnqueueRun status code | 201 | 201 | [ ] |
| ClaimRun status code | 200 | 200 | [ ] |
| Error status codes | Identical | Identical | [ ] |

---

### Error Scenarios
- [ ] 404 on non-existent task ID returns same status on both transports
- [ ] 400 on invalid JSON returns same status on both transports
- [ ] 409 on invalid status transition returns same status on both transports

---

### Related Test Cases
- TC-INT-001: HTTP POST /api/tasks baseline behavior
- TC-INT-002: HTTP GET /api/tasks list behavior
- TC-INT-003: HTTP GET /api/tasks/:id detail behavior
- TC-INT-004: HTTP PATCH immutable field rejection
