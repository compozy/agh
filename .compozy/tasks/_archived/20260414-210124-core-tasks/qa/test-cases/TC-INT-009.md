## TC-INT-009: Start run triggers SessionExecutor.StartTaskSession with dedicated session bound to run

**Priority:** P0
**Type:** Integration
**Status:** Not Run
**Estimated Time:** 10 minutes
**Created:** 2026-04-14

---

### Objective
Validate the session bridge integration: when a task run transitions through the start lifecycle (enqueue -> claim -> start), the TaskManager calls `SessionExecutor.StartTaskSession` with the correct `StartTaskSession` spec, a dedicated session is created, and the returned `session_id` is durably bound to the run record.

---

### Preconditions
- [ ] AGH daemon running with all subsystems including a functional SessionExecutor
- [ ] HTTP server listening on TCP :2123
- [ ] At least one agent configured and available for session creation
- [ ] A mock or real SessionExecutor implementation that tracks StartTaskSession calls
- [ ] One task T created (scope=global, title="Session Bridge Test", status=pending or ready)

---

### Test Steps

1. **Enqueue a run for the task**
   - Input:
     ```http
     POST http://localhost:2123/api/tasks/<T.id>/runs
     Content-Type: application/json

     {}
     ```
   - **Expected:** HTTP 201
   - **Expected:** Run R returned with `status = "queued"`, `session_id = ""`
   - Record R.id

2. **Claim the run**
   - Input:
     ```http
     POST http://localhost:2123/api/task-runs/<R.id>/claim
     Content-Type: application/json

     {}
     ```
   - **Expected:** HTTP 200
   - **Expected:** Run returned with `status = "claimed"`
   - **Expected:** `claimed_by` is populated with the claiming actor identity
   - **Expected:** `claimed_at` is a valid timestamp
   - **Expected:** `session_id` is still empty (session not yet started)

3. **Start the run (triggers SessionExecutor.StartTaskSession)**
   - Input:
     ```http
     POST http://localhost:2123/api/task-runs/<R.id>/start
     Content-Type: application/json

     {}
     ```
   - **Expected:** HTTP 200
   - **Expected:** Run returned with `status = "starting"` or `"running"` (depending on whether the session start is synchronous)
   - **Expected:** SessionExecutor.StartTaskSession was called with a `StartTaskSession` spec containing:
     - `task.id` = T.id
     - `task.title` = "Session Bridge Test"
     - `task.scope` = "global"
     - `run.id` = R.id
     - `run.task_id` = T.id
     - `run.status` = current run status at call time
     - `actor` = valid ActorContext
   - **Expected:** The returned `SessionRef.session_id` is non-empty

4. **Verify session_id is bound to the run**
   - Input: `GET http://localhost:2123/api/tasks/<T.id>` (or list runs)
   - **Expected:** The run in `task.runs` has `session_id` populated with the value from SessionRef
   - **Expected:** The `started_at` timestamp is set

5. **Verify the session actually exists**
   - Input: `GET http://localhost:2123/api/sessions/<session_id>`
   - **Expected:** HTTP 200
   - **Expected:** Session exists and is associated with the task context

6. **Complete the run and verify session lifecycle**
   - Input:
     ```http
     POST http://localhost:2123/api/task-runs/<R.id>/complete
     Content-Type: application/json

     {"result": {"output": "done"}}
     ```
   - **Expected:** HTTP 200
   - **Expected:** Run returned with `status = "completed"`, `ended_at` set, `result` contains the output
   - **Expected:** Task T status transitions to "completed" (if this was the successful run)

7. **Verify audit events include session binding**
   - Input: `GET http://localhost:2123/api/tasks/<T.id>` (check events)
   - **Expected:** Events include run_enqueued, run_claimed, run_started (with session binding info), run_completed

---

### Data Validation
| Field | Source Value | Expected Value | Status |
|-------|-------------|----------------|--------|
| Run status after enqueue | queued | "queued" | [ ] |
| Run session_id after enqueue | empty | "" | [ ] |
| Run status after claim | claimed | "claimed" | [ ] |
| Run claimed_by after claim | (populated) | Valid ActorIdentity | [ ] |
| Run status after start | starting/running | "starting" or "running" | [ ] |
| Run session_id after start | (populated) | Non-empty session UUID | [ ] |
| StartTaskSession spec.task.id | (passed to executor) | T.id | [ ] |
| StartTaskSession spec.run.id | (passed to executor) | R.id | [ ] |
| Session existence | GET /sessions/<id> | 200 | [ ] |
| Run status after complete | completed | "completed" | [ ] |
| Run ended_at after complete | (set) | Valid timestamp | [ ] |

---

### Error Scenarios
- [ ] Start run when SessionExecutor returns error: run transitions to failed, error recorded
- [ ] Start run on a task that is already cancelled: returns 409 (ErrInvalidStatusTransition)
- [ ] Start run that is not in "claimed" status: returns 409 (ErrInvalidStatusTransition)
- [ ] Enqueue run on a cancelled task: returns 409

---

### Related Test Cases
- TC-INT-010: AttachRunSession for the resume flow (alternative to StartTaskSession)
- TC-INT-008: Cancel propagation stops active sessions
- TC-INT-003: GetTask detail includes run with session binding
