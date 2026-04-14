## TC-FUNC-016: Start claimed run creates dedicated session

**Priority:** P0
**Type:** Functional
**Status:** Not Run
**Estimated Time:** 8 minutes
**Created:** 2026-04-14

---

### Objective
Validate that starting a claimed run transitions it through "starting" to "running" status, invokes the SessionExecutor to create a dedicated session, binds the session to the run, sets started_at timestamp, reconciles the parent task to "in_progress", and records both task.run_starting and task.run_started audit events.

---

### Preconditions
- [ ] AGH daemon running (or test harness initialized with TaskManager, backing Store, and mock SessionExecutor)
- [ ] One existing task with a claimed run (status="claimed")
- [ ] SessionExecutor mock configured to return a valid SessionRef
- [ ] ActorContext with Authority.Write=true

---

### Test Steps

1. **Start the claimed run**
   - Call StartRun(ctx, runID, StartRun{}, actor)
   - **Expected:** No error returned

2. **Verify SessionExecutor.StartTaskSession was called**
   - **Expected:** Called once with StartTaskSession containing the task, run, and actor context

3. **Inspect the returned TaskRun record**
   - **Expected:**
     - `run.Status` == "running"
     - `run.SessionID` is non-empty (bound to the session returned by SessionExecutor)
     - `run.StartedAt` is non-zero and close to now
     - `run.ClaimedBy` unchanged from claim step
     - `run.ClaimedAt` unchanged from claim step

4. **Verify the parent task reconciled to "in_progress"**
   - Read the task from store
   - **Expected:** Task status == "in_progress"

5. **Verify audit events were recorded**
   - Query events for the task
   - **Expected:**
     - TaskEvent with EventType="task.run_starting" (intermediate state)
     - TaskEvent with EventType="task.run_started" (final state)
     - Both events reference the run ID and session ID

6. **Test SessionExecutor failure during start**
   - Configure mock to return an error
   - Create a new claimed run, attempt to start it
   - **Expected:** Run transitions to "failed" status with error message containing the session failure; task reconciles accordingly

7. **Test SessionExecutor returns nil SessionRef**
   - Configure mock to return nil
   - **Expected:** Run transitions to "failed" with appropriate error

---

### Edge Cases & Variations

| Variation | Input | Expected Result |
|-----------|-------|-----------------|
| Start already-running run | run.Status="running" | ErrInvalidStatusTransition |
| Start a "starting" run (with session) | run.Status="starting", session_id set | Transitions to "running" (re-entry allowed) |
| Start a "starting" run (no session) | run.Status="starting", session_id="" | ErrInvalidStatusTransition (needs session binding) |
| SessionRef with empty session_id | SessionRef{SessionID:""} | Run fails; ErrValidation from SessionRef.Validate() |

---

### Related Test Cases
- TC-FUNC-015: Claim queued run
- TC-FUNC-017: Complete running run with result
- TC-FUNC-018: Fail running run with error
- TC-FUNC-020: Attach session to claimed run
