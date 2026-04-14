## TC-FUNC-025: Cancel already-terminal task (no-op or appropriate error)

**Priority:** P1
**Type:** Functional
**Status:** Not Run
**Estimated Time:** 3 minutes
**Created:** 2026-04-14

---

### Objective
Validate that attempting to cancel a task that is already in a terminal state ("completed", "failed", or "cancelled") either returns a no-op response or an appropriate error, and does not modify the task or its runs.

---

### Preconditions
- [ ] AGH daemon running (or test harness initialized with TaskManager and backing Store)
- [ ] Three tasks in terminal states:
  - Task A: status="completed", ClosedAt set
  - Task B: status="failed", ClosedAt set
  - Task C: status="cancelled", ClosedAt set
- [ ] ActorContext with Authority.Write=true

---

### Test Steps

1. **Attempt to cancel a completed task**
   - Call CancelTask(ctx, taskA.ID, CancelTask{Reason: "too late"}, actor)
   - **Expected:** Error returned (ErrInvalidStatusTransition or similar) OR no-op return; task status remains "completed"

2. **Verify completed task was not modified**
   - Read Task A from store
   - **Expected:** Status == "completed"; UpdatedAt unchanged; ClosedAt unchanged

3. **Attempt to cancel a failed task**
   - Call CancelTask(ctx, taskB.ID, CancelTask{Reason: "cleanup"}, actor)
   - **Expected:** Error returned OR no-op; task status remains "failed"

4. **Verify failed task was not modified**
   - Read Task B from store
   - **Expected:** Status == "failed"; all timestamps unchanged

5. **Attempt to cancel an already-cancelled task**
   - Call CancelTask(ctx, taskC.ID, CancelTask{Reason: "double cancel"}, actor)
   - **Expected:** Error returned OR idempotent no-op; task status remains "cancelled"

6. **Verify cancelled task was not modified**
   - Read Task C from store
   - **Expected:** Status == "cancelled"; all timestamps unchanged

7. **Verify no new audit events were recorded for any of these attempts**
   - **Expected:** No new task.cancelled events (or only one if idempotent)

---

### Edge Cases & Variations

| Variation | Input | Expected Result |
|-----------|-------|-----------------|
| Cancel completed task | CancelTask on completed | No-op or error; no state change |
| Cancel failed task | CancelTask on failed | No-op or error; no state change |
| Cancel cancelled task | CancelTask on cancelled | Idempotent no-op or error |
| Cancel in_progress task | CancelTask on in_progress | Should succeed (non-terminal) |
| Cancel pending task | CancelTask on pending | Should succeed (non-terminal) |

---

### Related Test Cases
- TC-FUNC-022: Cancel task with queued runs
- TC-FUNC-023: Cancel parent with running child runs
- TC-FUNC-024: Cancel propagation to grandchildren
