## TC-FUNC-030: Boot with orphaned running run (dead session) marked failed

**Priority:** P0
**Type:** Functional
**Status:** Not Run
**Estimated Time:** 8 minutes
**Created:** 2026-04-14

---

### Objective
Validate that during daemon boot recovery, a run discovered in "running" status with a dead/unreachable session is marked as failed via RunBootRecoveryFail action. The run transitions to "failed" with an error describing the orphaned-on-boot reason, ended_at is set, and a task.run_recovered audit event is recorded. The parent task reconciles accordingly.

---

### Preconditions
- [ ] Test harness initialized with TaskManager and backing Store
- [ ] Store pre-populated with:
  - One task (status="in_progress")
  - One run in "running" status with session_id="session-dead" (session process is gone after restart)
- [ ] ActorContext representing the daemon boot actor (Kind:"daemon", Ref:"agh-boot")
- [ ] SessionExecutor confirms the session is no longer alive

---

### Test Steps

1. **Call RecoverRunOnBoot with RunBootRecoveryFail action**
   - Input:
     ```json
     {
       "action": "fail",
       "reason": "orphaned running run with dead session discovered on boot",
       "session_state": "not_found"
     }
     ```
   - Call RecoverRunOnBoot(ctx, runID, RunBootRecovery{Action: "fail", Reason: "orphaned running run with dead session discovered on boot", SessionState: "not_found"}, actor)
   - **Expected:** No error returned

2. **Inspect the returned TaskRun record**
   - **Expected:**
     - `run.Status` == "failed"
     - `run.Error` is non-empty and contains "orphaned" or boot recovery context
     - `run.EndedAt` is non-zero and close to now
     - `run.SessionID` == "session-dead" (preserved for audit trail)
     - `run.Result` is nil (failure, not completion)

3. **Verify the run is persisted with failed status**
   - Read the run from store
   - **Expected:** All fields match step 2

4. **Verify the parent task reconciled**
   - Read the task from store
   - **Expected:** Task status reconciles to "failed" (if this was the only run) or stays "in_progress" (if other runs exist)

5. **Verify a task.run_recovered event was recorded**
   - Query events for the task
   - **Expected:** TaskEvent with EventType="task.run_recovered" containing:
     - action="fail" in payload
     - reason containing "orphaned" context
     - session_state="not_found"

6. **Test RunBootRecoveryMarkRunning for a starting run with live session**
   - Pre-populate a run in "starting" status with session_id="session-alive"
   - Call RecoverRunOnBoot with action="mark_running"
   - **Expected:** Run transitions to "running"; task reconciles to "in_progress"

7. **Verify that a new run can be enqueued on the task after recovery**
   - Enqueue a new run on the task
   - **Expected:** New run created successfully (task is not stuck in a bad state)

---

### Edge Cases & Variations

| Variation | Input | Expected Result |
|-----------|-------|-----------------|
| Running run with dead session | action="fail" | Run failed with orphan error |
| Starting run with dead session | action="fail" | Run failed with orphan error |
| Starting run with live session | action="mark_running" | Run promoted to "running" |
| Claimed run with no session | action="requeue" | Run re-queued (TC-FUNC-029 scenario) |
| Recovery on terminal run | action="fail" on already-completed run | Error or no-op |
| Recovery with empty reason | reason="" | Normalized to default reason |
| Multiple orphaned runs on boot | Two running runs from different tasks | Each recovered independently |

---

### Related Test Cases
- TC-FUNC-029: Boot with orphaned claimed run (no session) re-queued
- TC-FUNC-018: Fail running run with error
- TC-FUNC-016: Start claimed run creates dedicated session
