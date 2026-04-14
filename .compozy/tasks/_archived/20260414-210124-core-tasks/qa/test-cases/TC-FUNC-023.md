## TC-FUNC-023: Cancel parent with running child runs (cooperative stop then forced)

**Priority:** P0
**Type:** Functional
**Status:** Not Run
**Estimated Time:** 10 minutes
**Created:** 2026-04-14

---

### Objective
Validate that cancelling a parent task with active (running) child runs triggers cooperative session stop requests, waits for the grace period, and then force-stops sessions that did not terminate cooperatively. All runs and child tasks transition to "cancelled", and appropriate audit events are recorded.

---

### Preconditions
- [ ] AGH daemon running (or test harness initialized with TaskManager, backing Store, and mock SessionExecutor)
- [ ] Parent task with one child task
- [ ] Child task has a running run (status="running", session_id="session-child")
- [ ] SessionExecutor mock configured to track StopTaskSession calls
- [ ] CancelGracePeriod configured (e.g., 5 seconds for testing)
- [ ] ActorContext with Authority.Write=true

---

### Test Steps

1. **Cancel the parent task**
   - Call CancelTask(ctx, parentID, CancelTask{Reason: "project terminated"}, actor)
   - **Expected:** No error returned

2. **Verify the parent task is cancelled**
   - Read parent task
   - **Expected:** Status == "cancelled", ClosedAt set

3. **Verify the child task is cancelled**
   - Read child task
   - **Expected:** Status == "cancelled", ClosedAt set

4. **Verify the child's running run is cancelled**
   - Read child's runs
   - **Expected:** Run status == "cancelled", EndedAt set

5. **Verify cooperative stop was requested**
   - Check SessionExecutor mock
   - **Expected:** StopTaskSession called with reason="cancellation" for the child's session

6. **Verify audit events recorded**
   - **Expected:**
     - task.cancelled event on parent
     - task.cancelled event on child
     - task.run_cancelled event on child's run
     - Possibly task.run_force_stopped if the session did not stop cooperatively within grace period

7. **Test force-stop after grace period**
   - Configure SessionExecutor mock to delay/not respond to cooperative stop
   - Cancel a new parent with running child
   - **Expected:** After grace period, force stop is triggered; task.run_force_stopped event recorded

---

### Edge Cases & Variations

| Variation | Input | Expected Result |
|-----------|-------|-----------------|
| Child cooperatively stops within grace period | Session stops before timeout | Clean cancellation; no force-stop event |
| Child does not stop within grace period | Session ignores stop | Force-stop triggered after grace period |
| Multiple running children | Parent has 3 children, each with running runs | All 3 sessions receive stop requests |
| Child has queued run (no session) | Run status="queued" | Run cancelled immediately; no session stop needed |
| Child has claimed run (no session) | Run status="claimed" | Run cancelled immediately |

---

### Related Test Cases
- TC-FUNC-022: Cancel task with queued runs
- TC-FUNC-024: Cancel propagation to grandchildren
- TC-FUNC-025: Cancel already-terminal task
