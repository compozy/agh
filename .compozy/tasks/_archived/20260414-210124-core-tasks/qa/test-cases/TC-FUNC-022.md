## TC-FUNC-022: Cancel task with queued runs

**Priority:** P0
**Type:** Functional
**Status:** Not Run
**Estimated Time:** 5 minutes
**Created:** 2026-04-14

---

### Objective
Validate that cancelling a task with queued (non-started) runs immediately transitions the runs to "cancelled", transitions the task to "cancelled", records task.cancelled and task.run_cancelled audit events, and sets appropriate timestamps.

---

### Preconditions
- [ ] AGH daemon running (or test harness initialized with TaskManager and backing Store)
- [ ] One existing task with two queued runs (run-A status="queued", run-B status="queued")
- [ ] ActorContext with Authority.Write=true

---

### Test Steps

1. **Cancel the task**
   - Input:
     ```json
     {
       "reason": "Requirements changed"
     }
     ```
   - Call CancelTask(ctx, taskID, CancelTask{Reason: "Requirements changed"}, actor)
   - **Expected:** No error returned

2. **Inspect the returned Task record**
   - **Expected:**
     - `task.Status` == "cancelled"
     - `task.ClosedAt` is non-zero and close to now
     - `task.UpdatedAt` >= original UpdatedAt

3. **Verify all queued runs are cancelled**
   - Query runs for the task
   - **Expected:**
     - run-A status == "cancelled", EndedAt set
     - run-B status == "cancelled", EndedAt set

4. **Verify audit events were recorded**
   - Query events for the task
   - **Expected:**
     - TaskEvent with EventType="task.cancelled"
     - TaskEvent with EventType="task.run_cancelled" for run-A
     - TaskEvent with EventType="task.run_cancelled" for run-B

5. **Verify no session stop was requested for queued runs**
   - **Expected:** SessionExecutor.StopTaskSession was NOT called (queued runs have no sessions)

---

### Edge Cases & Variations

| Variation | Input | Expected Result |
|-----------|-------|-----------------|
| Cancel task with no runs | Task has no runs | Task cancelled; no run events |
| Cancel task with mix of queued and claimed | One queued, one claimed | Both cancelled immediately (no session to stop) |
| Cancel with reason | CancelTask{Reason:"budget cut"} | Reason stored in event payload |
| Cancel with metadata | CancelTask{Metadata: json} | Metadata stored; validated against payload size limit |

---

### Related Test Cases
- TC-FUNC-023: Cancel parent with running child runs
- TC-FUNC-024: Cancel propagation to grandchildren
- TC-FUNC-025: Cancel already-terminal task
