## TC-FUNC-014: Enqueue run on ready task

**Priority:** P0
**Type:** Functional
**Status:** Not Run
**Estimated Time:** 5 minutes
**Created:** 2026-04-14

---

### Objective
Validate that enqueueing a run on a task in "ready" (or "pending" with no deps) status creates a TaskRun record with status "queued", attempt=1, correct origin, and records a task.run_enqueued audit event.

---

### Preconditions
- [ ] AGH daemon running (or test harness initialized with TaskManager and backing Store)
- [ ] One existing task with status="pending" and no dependencies (will reconcile to ready or be treated as executable)
- [ ] ActorContext with Authority.Write=true

---

### Test Steps

1. **Enqueue a run on the task**
   - Input:
     ```json
     {
       "task_id": "<task-id>"
     }
     ```
   - **Expected:** No error returned

2. **Inspect the returned TaskRun record**
   - **Expected:**
     - `run.ID` is non-empty and server-generated
     - `run.TaskID` == task ID
     - `run.Status` == "queued"
     - `run.Attempt` == 1
     - `run.Origin` matches actor's origin
     - `run.QueuedAt` is non-zero and close to now
     - `run.ClaimedBy` == nil
     - `run.ClaimedAt` is zero
     - `run.SessionID` == ""
     - `run.StartedAt` is zero
     - `run.EndedAt` is zero
     - `run.Result` is nil/empty
     - `run.Error` == ""

3. **Verify the run is persisted in the store**
   - Query runs for the task
   - **Expected:** One run matching all fields from step 2

4. **Verify a task.run_enqueued event was recorded**
   - Query events for the task
   - **Expected:** TaskEvent with EventType="task.run_enqueued", RunID=run.ID

5. **Enqueue a second run on the same task**
   - **Expected:** Second run created with Attempt=2 (or new attempt number), separate ID

---

### Edge Cases & Variations

| Variation | Input | Expected Result |
|-----------|-------|-----------------|
| Enqueue on blocked task | Task status="blocked" | Rejected (task not executable) |
| Enqueue on completed task | Task status="completed" | Rejected |
| Enqueue on cancelled task | Task status="cancelled" | Rejected |
| Enqueue with idempotency_key | idempotency_key="key-1" | Run created with idempotency tracking |
| Enqueue with network_channel | network_channel="chan-1" | Run created with channel binding |
| Empty task_id | task_id="" | ErrValidation |

---

### Related Test Cases
- TC-FUNC-015: Claim queued run
- TC-FUNC-019: Invalid transition (queued to running)
- TC-FUNC-021: Idempotent run enqueue
