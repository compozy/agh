## TC-INT-010: AttachRunSession binds existing session; second attach returns ErrSessionAlreadyBound

**Priority:** P0
**Type:** Integration
**Status:** Not Run
**Estimated Time:** 8 minutes
**Created:** 2026-04-14

---

### Objective
Validate the session resume flow: `AttachRunSession` binds an existing session to a claimed/starting run, and a second attach attempt on the same run returns `ErrSessionAlreadyBound` (HTTP 409). Also verify that the `SessionExecutor.AttachTaskSession` is called with the correct parameters.

---

### Preconditions
- [ ] AGH daemon running with all subsystems including a functional SessionExecutor
- [ ] HTTP server listening on TCP :2123
- [ ] An existing session S1 created independently (via POST /api/sessions or agent launch)
- [ ] A second session S2 created independently
- [ ] One task T created (scope=global, title="Attach Session Test")
- [ ] One run R enqueued and claimed for task T (status="claimed")

---

### Test Steps

1. **Attach session S1 to the claimed run**
   - Input:
     ```http
     POST http://localhost:2123/api/task-runs/<R.id>/attach-session
     Content-Type: application/json

     {
       "session_id": "<S1.id>"
     }
     ```
   - **Expected:** HTTP 200
   - **Expected:** Run returned with `session_id` equal to S1.id
   - **Expected:** `SessionExecutor.AttachTaskSession` was called with `runID = R.id` and `sessionID = S1.id`
   - **Expected:** The returned SessionRef is valid

2. **Verify the session binding persists**
   - Input: `GET http://localhost:2123/api/tasks/<T.id>`
   - **Expected:** HTTP 200
   - **Expected:** The run in `task.runs` has `session_id = S1.id`

3. **Attempt to attach session S2 to the same run (already bound)**
   - Input:
     ```http
     POST http://localhost:2123/api/task-runs/<R.id>/attach-session
     Content-Type: application/json

     {
       "session_id": "<S2.id>"
     }
     ```
   - **Expected:** HTTP 409 Conflict
   - **Expected:** Error message references ErrSessionAlreadyBound
   - **Expected:** The run's session_id remains S1.id (unchanged)

4. **Attempt to re-attach the same session S1 to the same run**
   - Input:
     ```http
     POST http://localhost:2123/api/task-runs/<R.id>/attach-session
     Content-Type: application/json

     {
       "session_id": "<S1.id>"
     }
     ```
   - **Expected:** HTTP 409 Conflict (session is already bound, even if it's the same session)

5. **Verify run state is not corrupted after failed attach attempts**
   - Input: `GET http://localhost:2123/api/tasks/<T.id>`
   - **Expected:** HTTP 200
   - **Expected:** Run still has `session_id = S1.id`
   - **Expected:** Run status is still "claimed" or "starting" (not corrupted)

6. **Attach session to a run in "starting" status**
   - Create new task T2, enqueue run R2, claim R2, then start R2 (status="starting")
   - If R2 was started without a session (manual start flow):
     ```http
     POST http://localhost:2123/api/task-runs/<R2.id>/attach-session
     Content-Type: application/json

     {
       "session_id": "<S2.id>"
     }
     ```
   - **Expected:** HTTP 200 if the run is in a state that allows attachment
   - **Expected:** Or HTTP 409 (ErrSessionAttachNotAllowed) if the run already has a session from the start flow

7. **Attach session to a completed run**
   - Create a run that has been completed
   - Input:
     ```http
     POST http://localhost:2123/api/task-runs/<completed_run_id>/attach-session
     Content-Type: application/json

     {
       "session_id": "<S2.id>"
     }
     ```
   - **Expected:** HTTP 409 (ErrSessionAttachNotAllowed -- cannot attach to a terminal run)

8. **Attach with empty session_id**
   - Input:
     ```http
     POST http://localhost:2123/api/task-runs/<R.id>/attach-session
     Content-Type: application/json

     {
       "session_id": ""
     }
     ```
   - **Expected:** HTTP 400 (ErrValidation: session_id is required)

---

### Data Validation
| Field | Source Value | Expected Value | Status |
|-------|-------------|----------------|--------|
| Run session_id after first attach | S1.id | S1.id | [ ] |
| HTTP status (first attach) | Response code | 200 | [ ] |
| HTTP status (second attach, different session) | Response code | 409 | [ ] |
| HTTP status (re-attach same session) | Response code | 409 | [ ] |
| Run session_id after failed attach | S1.id | S1.id (unchanged) | [ ] |
| HTTP status (attach to completed run) | Response code | 409 | [ ] |
| HTTP status (empty session_id) | Response code | 400 | [ ] |
| Error type (second attach) | Error | ErrSessionAlreadyBound | [ ] |
| Error type (terminal run) | Error | ErrSessionAttachNotAllowed | [ ] |

---

### Error Scenarios
- [ ] Attach to non-existent run ID returns 404 (ErrTaskRunNotFound)
- [ ] Attach with non-existent session ID returns 404 (session not found) or appropriate error
- [ ] Attach to a "queued" run returns 409 (ErrSessionAttachNotAllowed -- must be claimed or starting first)
- [ ] Attach to a "cancelled" run returns 409 (ErrSessionAttachNotAllowed)
- [ ] Attach to a "failed" run returns 409 (ErrSessionAttachNotAllowed)

---

### Related Test Cases
- TC-INT-009: StartTaskSession flow (creates session automatically via bridge)
- TC-INT-008: Cancel propagation affects runs with attached sessions
- TC-INT-005: UDS parity for attach-session endpoint
