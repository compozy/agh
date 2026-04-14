## SMOKE-006: Enqueue and Claim a Task Run

**Priority:** P0
**Type:** Smoke
**Status:** Not Run
**Estimated Time:** 2-3 minutes
**Created:** 2026-04-14

---

### Objective
Quick sanity check that a task run can be enqueued and then claimed, transitioning through the expected lifecycle states: queued -> claimed.

---

### Preconditions
- [ ] AGH daemon running with task subsystem initialized
- [ ] At least one task exists in `pending` or `ready` status

---

### Test Steps
1. **Enqueue a run for an existing task**
   - Input: `POST /api/tasks/<task-id>/runs` with body:
     ```json
     {
       "idempotency_key": "smoke-run-001"
     }
     ```
   - **Expected:** 201 Created (or 200 OK). Response includes:
     - `id`: non-empty run ID (e.g., `"run_..."`)
     - `task_id`: matches the parent task ID
     - `status`: `"queued"`
     - `attempt`: 1
     - `queued_at`: valid timestamp
     - `origin`: server-derived

2. **Verify task status updated**
   - Input: `GET /api/tasks/<task-id>`
   - **Expected:** Task status may have transitioned (e.g., to `"in_progress"` or remain `"ready"` depending on run state).

3. **Claim the run**
   - Input: `POST /api/task-runs/<run-id>/claim` with body `{}`
   - **Expected:** 200 OK. Response shows:
     - `status`: `"claimed"`
     - `claimed_by`: populated with the authenticated principal identity
     - `claimed_at`: valid timestamp

4. **Verify run appears in task detail**
   - Input: `GET /api/tasks/<task-id>`
   - **Expected:** `runs` array contains the run with status `"claimed"`.

5. **Verify audit events**
   - Input: Check events in task detail
   - **Expected:** Events include `"task.run_enqueued"` and `"task.run_claimed"` entries.

---

### Related Test Cases
- SMOKE-007: Start and complete a run
- SMOKE-008: Cancel a task
- TC-PERF-001: Task creation throughput
