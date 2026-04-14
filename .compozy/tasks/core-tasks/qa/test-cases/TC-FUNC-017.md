## TC-FUNC-017: Complete running run with result

**Priority:** P0
**Type:** Functional
**Status:** Not Run
**Estimated Time:** 5 minutes
**Created:** 2026-04-14

---

### Objective
Validate that completing a running task run transitions it to "completed" status, persists the result_json payload, sets ended_at timestamp, reconciles the parent task status (potentially to "completed"), and records a task.run_completed audit event.

---

### Preconditions
- [ ] AGH daemon running (or test harness initialized with TaskManager and backing Store)
- [ ] One existing task with a running run (status="running", session_id set)
- [ ] ActorContext with Authority.Write=true

---

### Test Steps

1. **Complete the running run with a result**
   - Input:
     ```json
     {
       "value": {"output": "deployment successful", "duration_ms": 1234}
     }
     ```
   - Call CompleteRun(ctx, runID, RunResult{Value: resultJSON}, actor)
   - **Expected:** No error returned

2. **Inspect the returned TaskRun record**
   - **Expected:**
     - `run.Status` == "completed"
     - `run.Result` contains the provided JSON result
     - `run.EndedAt` is non-zero and close to now
     - `run.Error` == "" (no error on successful completion)
     - `run.SessionID` unchanged
     - `run.StartedAt` < `run.EndedAt`

3. **Verify the parent task reconciled**
   - Read the task from store
   - **Expected:** If this was the only run, task status reconciles to "completed"

4. **Verify a task.run_completed event was recorded**
   - Query events for the task
   - **Expected:** TaskEvent with EventType="task.run_completed", RunID=run.ID

5. **Complete the run with empty/nil result**
   - Create another running run; complete with RunResult{Value: nil}
   - **Expected:** Run completes successfully with nil result stored

---

### Edge Cases & Variations

| Variation | Input | Expected Result |
|-----------|-------|-----------------|
| Complete with large valid result (under 64KB) | 60KB JSON | Success |
| Complete with result exceeding 64KB | 65KB JSON | ErrPayloadTooLarge |
| Complete non-running run (queued) | run.Status="queued" | ErrInvalidStatusTransition |
| Complete non-running run (claimed) | run.Status="claimed" | ErrInvalidStatusTransition |
| Complete already-completed run | run.Status="completed" | ErrInvalidStatusTransition |
| Complete with invalid JSON in result | result=`{broken` | ErrValidation |

---

### Related Test Cases
- TC-FUNC-016: Start claimed run
- TC-FUNC-018: Fail running run with error
- TC-FUNC-027: Complete run with result_json > 64KB
