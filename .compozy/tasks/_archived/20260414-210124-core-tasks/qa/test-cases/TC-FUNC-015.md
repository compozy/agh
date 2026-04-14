## TC-FUNC-015: Claim queued run

**Priority:** P0
**Type:** Functional
**Status:** Not Run
**Estimated Time:** 5 minutes
**Created:** 2026-04-14

---

### Objective
Validate that claiming a queued run transitions it to status "claimed", sets claimed_by to the actor identity, sets claimed_at timestamp, and records a task.run_claimed audit event.

---

### Preconditions
- [ ] AGH daemon running (or test harness initialized with TaskManager and backing Store)
- [ ] One existing task with a queued run (status="queued")
- [ ] ActorContext with Authority.Write=true, Actor={Kind:"agent_session", Ref:"session-42"}

---

### Test Steps

1. **Claim the queued run**
   - Call ClaimRun(ctx, runID, ClaimRun{}, actor)
   - **Expected:** No error returned

2. **Inspect the returned TaskRun record**
   - **Expected:**
     - `run.Status` == "claimed"
     - `run.ClaimedBy` == {Kind:"agent_session", Ref:"session-42"}
     - `run.ClaimedAt` is non-zero and close to now
     - `run.SessionID` still == "" (session attached later)
     - `run.StartedAt` still zero
     - All other fields unchanged from queued state

3. **Verify the claim is persisted in the store**
   - Read the run back by ID
   - **Expected:** All fields match step 2

4. **Verify a task.run_claimed event was recorded**
   - Query events for the task
   - **Expected:** TaskEvent with EventType="task.run_claimed", RunID=run.ID

5. **Attempt to claim the same run again**
   - Call ClaimRun(ctx, runID, ClaimRun{}, actor)
   - **Expected:** Error returned; ErrInvalidStatusTransition (run is already claimed, not queued)

6. **Attempt to claim a non-existent run**
   - Call ClaimRun(ctx, "nonexistent", ClaimRun{}, actor)
   - **Expected:** ErrTaskRunNotFound

---

### Edge Cases & Variations

| Variation | Input | Expected Result |
|-----------|-------|-----------------|
| Claim by human actor | Actor={Kind:"human", Ref:"user-1"} | claimed_by reflects human actor |
| Claim by daemon | Actor={Kind:"daemon", Ref:"agh"} | claimed_by reflects daemon actor |
| Claim with idempotency_key | ClaimRun{IdempotencyKey:"key-1"} | Idempotency tracked |
| Claim run on non-executable task | Task became cancelled between enqueue and claim | Rejected |

---

### Related Test Cases
- TC-FUNC-014: Enqueue run on ready task
- TC-FUNC-016: Start claimed run
- TC-FUNC-019: Invalid transition (queued to running)
