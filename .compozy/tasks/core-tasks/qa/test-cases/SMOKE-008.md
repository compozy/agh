## SMOKE-008: Cancel a Task and Its Active Runs

**Priority:** P0
**Type:** Smoke
**Status:** Not Run
**Estimated Time:** 2-3 minutes
**Created:** 2026-04-14

---

### Objective
Quick sanity check that cancelling a task transitions both the task and its active (non-terminal) runs to cancelled status, with proper audit events and reason propagation.

---

### Preconditions
- [ ] AGH daemon running with task subsystem initialized
- [ ] A task exists with at least one active run (queued or claimed)

---

### Test Steps
1. **Set up: create task with an active run**
   - Input: Create a task, enqueue a run, optionally claim it
   - **Expected:** Task in `ready` or `in_progress` status. Run in `queued` or `claimed` status.

2. **Cancel the task**
   - Input: `POST /api/tasks/<task-id>/cancel` with body:
     ```json
     {
       "reason": "Smoke test cancellation"
     }
     ```
   - **Expected:** 200 OK. Response shows task with `status: "cancelled"`.

3. **Verify task is cancelled**
   - Input: `GET /api/tasks/<task-id>`
   - **Expected:** Task `status` is `"cancelled"`. `closed_at` is populated.

4. **Verify active runs are cancelled**
   - Input: Check `runs` array in task detail
   - **Expected:** All previously active runs now have `status: "cancelled"`. `ended_at` is populated.

5. **Verify audit events**
   - Input: Check events in task detail
   - **Expected:** Events include `"task.cancelled"` and `"task.run_cancelled"` entries. The `"task.cancelled"` event payload includes the reason `"Smoke test cancellation"`.

6. **Verify cancelled task cannot be re-activated**
   - Input: Attempt to enqueue a new run on the cancelled task
   - **Expected:** Error returned (invalid status transition). No new run created.

---

### Related Test Cases
- SMOKE-006: Enqueue and claim a run
- SMOKE-007: Start and complete a run
- TC-PERF-004: Cancellation propagation on large tree
