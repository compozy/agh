## TC-FUNC-020: Attach session to claimed run; attach again returns ErrSessionAlreadyBound

**Priority:** P0
**Type:** Functional
**Status:** Not Run
**Estimated Time:** 5 minutes
**Created:** 2026-04-14

---

### Objective
Validate that attaching an existing session to a claimed run binds the session_id, transitions the run to "starting", records a task.run_session_bound audit event, and that attempting to attach a second session to the same run is rejected with ErrSessionAlreadyBound.

---

### Preconditions
- [ ] AGH daemon running (or test harness initialized with TaskManager, backing Store, and mock SessionExecutor)
- [ ] One existing task with a claimed run (status="claimed", session_id="")
- [ ] SessionExecutor mock configured to return a valid SessionRef for AttachTaskSession
- [ ] ActorContext with Authority.Write=true

---

### Test Steps

1. **Attach a session to the claimed run**
   - Call AttachRunSession(ctx, runID, "session-abc", actor)
   - **Expected:** No error returned

2. **Inspect the returned TaskRun record**
   - **Expected:**
     - `run.SessionID` == "session-abc"
     - `run.Status` == "starting" (claimed -> starting on session bind)
     - Other fields (ClaimedBy, ClaimedAt, Attempt) unchanged

3. **Verify a task.run_session_bound event was recorded**
   - Query events for the task
   - **Expected:** TaskEvent with EventType="task.run_session_bound", RunID=run.ID

4. **Attempt to attach another session to the same run**
   - Call AttachRunSession(ctx, runID, "session-def", actor)
   - **Expected:** Error returned; `errors.Is(err, ErrSessionAlreadyBound)` == true

5. **Verify the run's session_id was not changed**
   - Read the run from store
   - **Expected:** `run.SessionID` still == "session-abc"

6. **Attempt to attach a session to a running run**
   - Start the run (moving to "running"), then call AttachRunSession
   - **Expected:** ErrSessionAttachNotAllowed (only claimed/starting runs accept attach)

---

### Edge Cases & Variations

| Variation | Input | Expected Result |
|-----------|-------|-----------------|
| Attach to claimed run | Valid | run transitions to "starting", session bound |
| Attach to starting run (no session) | run.Status="starting", session_id="" | Session bound successfully |
| Attach to starting run (with session) | run.Status="starting", session_id="existing" | ErrSessionAlreadyBound |
| Attach to queued run | run.Status="queued" | ErrSessionAttachNotAllowed |
| Attach to completed run | run.Status="completed" | ErrSessionAttachNotAllowed |
| Attach empty session_id | session_id="" | ErrValidation (session id required) |
| Attach whitespace session_id | session_id="   " | ErrValidation |
| Session already bound to another run | session-abc active on run-1, try on run-2 | ErrSessionAlreadyBound |

---

### Related Test Cases
- TC-FUNC-015: Claim queued run
- TC-FUNC-016: Start claimed run
- TC-FUNC-019: Invalid transition
