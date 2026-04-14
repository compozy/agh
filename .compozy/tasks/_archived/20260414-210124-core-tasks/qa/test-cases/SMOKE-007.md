## SMOKE-007: Start and Complete a Task Run

**Priority:** P0
**Type:** Smoke
**Status:** Not Run
**Estimated Time:** 2-3 minutes
**Created:** 2026-04-14

---

### Objective
Quick sanity check that a claimed task run can progress through the full lifecycle: claimed -> starting -> running -> completed. Validates the run state machine transitions and task status reconciliation.

---

### Preconditions
- [ ] AGH daemon running with task subsystem initialized
- [ ] A task run exists in `claimed` status (from SMOKE-006 or fresh setup)

---

### Test Steps
1. **Start the claimed run**
   - Input: `POST /api/task-runs/<run-id>/start` with body `{}`
   - **Expected:** 200 OK. Response shows:
     - `status`: `"starting"` (transitional) or `"running"` (if session starts immediately)
     - `started_at`: populated once running

2. **Verify task transitions to in_progress**
   - Input: `GET /api/tasks/<task-id>`
   - **Expected:** Task status is `"in_progress"`.

3. **Complete the run**
   - Input: `POST /api/task-runs/<run-id>/complete` with body:
     ```json
     {
       "result": {"output": "smoke test passed"}
     }
     ```
   - **Expected:** 200 OK. Response shows:
     - `status`: `"completed"`
     - `ended_at`: valid timestamp
     - `result`: `{"output": "smoke test passed"}`

4. **Verify task status after run completion**
   - Input: `GET /api/tasks/<task-id>`
   - **Expected:** Task status is `"completed"`. `closed_at` is populated.

5. **Verify audit trail**
   - Input: Check events in task detail
   - **Expected:** Events include `"task.run_started"` (or `"task.run_starting"`) and `"task.run_completed"` entries with correct timestamps.

---

### Related Test Cases
- SMOKE-006: Enqueue and claim a run
- SMOKE-008: Cancel a task
- TC-PERF-004: Cancellation propagation
