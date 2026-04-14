## TC-FUNC-018: Fail running run with error

**Priority:** P0
**Type:** Functional
**Status:** Not Run
**Estimated Time:** 5 minutes
**Created:** 2026-04-14

---

### Objective
Validate that failing a running task run transitions it to "failed" status, persists the error message and optional metadata, sets ended_at timestamp, reconciles the parent task status (potentially to "failed"), and records a task.run_failed audit event.

---

### Preconditions
- [ ] AGH daemon running (or test harness initialized with TaskManager and backing Store)
- [ ] One existing task with a running run (status="running", session_id set)
- [ ] ActorContext with Authority.Write=true

---

### Test Steps

1. **Fail the running run with an error**
   - Input:
     ```json
     {
       "error": "out of memory during build step",
       "metadata": {"exit_code": 137}
     }
     ```
   - Call FailRun(ctx, runID, RunFailure{Error: "out of memory during build step", Metadata: metadataJSON}, actor)
   - **Expected:** No error returned

2. **Inspect the returned TaskRun record**
   - **Expected:**
     - `run.Status` == "failed"
     - `run.Error` == "out of memory during build step"
     - `run.EndedAt` is non-zero and close to now
     - `run.Result` is nil/empty (failure, not completion)
     - `run.SessionID` unchanged

3. **Verify the parent task reconciled**
   - Read the task from store
   - **Expected:** Task status reconciles to "failed" (if no other active runs)

4. **Verify a task.run_failed event was recorded**
   - Query events for the task
   - **Expected:** TaskEvent with EventType="task.run_failed", RunID=run.ID

5. **Attempt to fail with empty error message**
   - Input: RunFailure{Error: ""}
   - **Expected:** ErrValidation returned; error field is required

---

### Edge Cases & Variations

| Variation | Input | Expected Result |
|-----------|-------|-----------------|
| Fail with only error (no metadata) | RunFailure{Error: "timeout"} | Success; metadata is nil |
| Fail with large metadata | metadata > 64KB | ErrPayloadTooLarge |
| Fail non-running run (queued) | run.Status="queued" | ErrInvalidStatusTransition |
| Fail non-running run (completed) | run.Status="completed" | ErrInvalidStatusTransition |
| Fail with whitespace-only error | RunFailure{Error: "   "} | ErrValidation (error required after trim) |
| Multiple runs: fail one, task has another running | Fail run-1, run-2 still running | Task stays "in_progress" |

---

### Related Test Cases
- TC-FUNC-016: Start claimed run
- TC-FUNC-017: Complete running run with result
- TC-FUNC-019: Invalid transition (queued to running)
