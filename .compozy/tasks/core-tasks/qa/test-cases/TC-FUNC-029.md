## TC-FUNC-029: Boot with orphaned claimed run (no session) re-queued

**Priority:** P0
**Type:** Functional
**Status:** Not Run
**Estimated Time:** 8 minutes
**Created:** 2026-04-14

---

### Objective
Validate that during daemon boot recovery, a run discovered in "claimed" status with no session binding (session_id="") is re-queued via RunBootRecoveryRequeue action. The run transitions back to "queued", claimed_by is cleared, and a task.run_recovered audit event is recorded with the recovery reason.

---

### Preconditions
- [ ] Test harness initialized with TaskManager and backing Store
- [ ] Store pre-populated with:
  - One task (status="in_progress")
  - One run in "claimed" status with session_id="" (orphaned -- claimed but never started, daemon crashed)
- [ ] ActorContext representing the daemon boot actor (Kind:"daemon", Ref:"agh-boot")

---

### Test Steps

1. **Call RecoverRunOnBoot with RunBootRecoveryRequeue action**
   - Input:
     ```json
     {
       "action": "requeue",
       "reason": "orphaned claimed run discovered on boot"
     }
     ```
   - Call RecoverRunOnBoot(ctx, runID, RunBootRecovery{Action: "requeue", Reason: "orphaned claimed run discovered on boot"}, actor)
   - **Expected:** No error returned

2. **Inspect the returned TaskRun record**
   - **Expected:**
     - `run.Status` == "queued"
     - `run.ClaimedBy` cleared (nil)
     - `run.ClaimedAt` reset or zeroed
     - `run.SessionID` == "" (unchanged, was already empty)
     - `run.Attempt` unchanged

3. **Verify the run is persisted with queued status**
   - Read the run from store
   - **Expected:** All fields match step 2

4. **Verify the parent task reconciled**
   - Read the task from store
   - **Expected:** Task status reconciles appropriately (may go back to "ready" or "pending" if no other active runs)

5. **Verify a task.run_recovered event was recorded**
   - Query events for the task
   - **Expected:** TaskEvent with EventType="task.run_recovered" containing the recovery action and reason in the payload

6. **Verify the run can be re-claimed after recovery**
   - Claim the re-queued run
   - **Expected:** Run transitions to "claimed" successfully

---

### Edge Cases & Variations

| Variation | Input | Expected Result |
|-----------|-------|-----------------|
| Claimed run with no session | action="requeue" | Run re-queued |
| Claimed run with session (should not happen) | action="requeue" | Depends on implementation; may still requeue or reject |
| Recovery with empty reason | reason="" | Normalized to default reason |
| Invalid recovery action | action="unknown" | ErrValidation |
| Run already in terminal state | action="requeue" on completed run | Error or no-op |

---

### Related Test Cases
- TC-FUNC-030: Boot with orphaned running run (dead session)
- TC-FUNC-015: Claim queued run
- TC-FUNC-014: Enqueue run on ready task
