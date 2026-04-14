## TC-INT-003: HTTP GET /api/v1/tasks/:id returns 200 with full TaskDetailPayload

**Priority:** P0
**Type:** Integration
**Status:** Not Run
**Estimated Time:** 8 minutes
**Created:** 2026-04-14

---

### Objective
Validate that HTTP GET `/api/tasks/:id` returns a complete TaskDetailPayload containing the task record, its children, dependencies, runs, and audit events, all in the correct contract shape.

---

### Preconditions
- [ ] AGH daemon running with all subsystems
- [ ] HTTP server listening on TCP :2123
- [ ] Seed data prepared:
  - Parent task P created (scope=global, title="Parent Task P")
  - Child task C1 created under P (via POST /api/tasks/<P.id>/children)
  - Child task C2 created under P
  - Dependency: P depends on an independent task D1 (via POST /api/tasks/<P.id>/dependencies)
  - Run R1 enqueued for P (via POST /api/tasks/<P.id>/runs)
  - Run R1 claimed, started (lifecycle progressed to generate events)

---

### Test Steps

1. **GET the parent task with full detail**
   - Input: `GET http://localhost:2123/api/tasks/<P.id>`
   - **Expected:** HTTP 200
   - **Expected:** Response shape is `{"task": <TaskDetailPayload>}`

2. **Validate the task object within the detail payload**
   - **Expected:** `task.task.id` equals P.id
   - **Expected:** `task.task.scope` equals `"global"`
   - **Expected:** `task.task.title` equals `"Parent Task P"`
   - **Expected:** `task.task.status` is a valid TaskStatus
   - **Expected:** `task.task.created_by` is a valid ActorIdentity
   - **Expected:** `task.task.origin` is a valid Origin
   - **Expected:** `task.task.created_at` and `task.task.updated_at` are valid timestamps

3. **Validate the children array**
   - **Expected:** `task.children` is an array with exactly 2 entries
   - **Expected:** Each child has `parent_task_id` equal to P.id
   - **Expected:** C1 and C2 IDs are present in the children list
   - **Expected:** Each child entry is a TaskSummaryPayload (contains id, scope, status, title, created_by, origin, but no description or metadata)

4. **Validate the dependencies array**
   - **Expected:** `task.dependencies` is an array with at least 1 entry
   - **Expected:** Entry contains `task_id` equal to P.id and `depends_on_task_id` equal to D1.id
   - **Expected:** `kind` equals `"blocks"` (default dependency kind)
   - **Expected:** `created_at` is a valid timestamp

5. **Validate the runs array**
   - **Expected:** `task.runs` is an array with at least 1 entry (R1)
   - **Expected:** R1 entry contains `id`, `task_id` (equals P.id), `status`, `attempt` (equals 1), `queued_at`
   - **Expected:** If R1 was claimed: `claimed_by` is populated, `claimed_at` is a valid timestamp
   - **Expected:** If R1 was started: `started_at` is a valid timestamp
   - **Expected:** `origin` is a valid Origin

6. **Validate the events array**
   - **Expected:** `task.events` is an array with multiple entries (task_created, run_enqueued, run_claimed, run_started at minimum)
   - **Expected:** Each event has `id`, `task_id` (equals P.id), `event_type`, `actor`, `origin`, `timestamp`
   - **Expected:** Events are ordered by timestamp (ascending or consistent ordering)
   - **Expected:** Run-related events have `run_id` set to R1.id

7. **GET a non-existent task**
   - Input: `GET http://localhost:2123/api/tasks/nonexistent-uuid-here`
   - **Expected:** HTTP 404
   - **Expected:** Response body contains error message referencing ErrTaskNotFound

8. **GET with empty ID path parameter**
   - Input: `GET http://localhost:2123/api/tasks/`
   - **Expected:** HTTP 400 or 404 (validation: task id is required)

---

### Data Validation
| Field | Source Value | Expected Value | Status |
|-------|-------------|----------------|--------|
| HTTP status (valid ID) | Response code | 200 | [ ] |
| task.task.id | P.id | Matches created task ID | [ ] |
| task.children | Array | Length 2, both match C1/C2 | [ ] |
| task.dependencies | Array | Length >= 1, contains D1 edge | [ ] |
| task.runs | Array | Length >= 1, contains R1 | [ ] |
| task.events | Array | Length >= 4 (create + run lifecycle) | [ ] |
| HTTP status (missing ID) | Response code | 404 | [ ] |

---

### Error Scenarios
- [ ] Non-existent task ID returns 404 (ErrTaskNotFound)
- [ ] Malformed or empty path parameter returns 400 or 404

---

### Related Test Cases
- TC-INT-001: Creates the parent task
- TC-INT-002: Lists tasks that should include the parent
- TC-INT-004: Tests immutable field update on the same task
