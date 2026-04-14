## SMOKE-010: Observe Projections Return Task Metrics

**Priority:** P0
**Type:** Smoke
**Status:** Not Run
**Estimated Time:** 2-3 minutes
**Created:** 2026-04-14

---

### Objective
Quick sanity check that the observe layer's task projection endpoints return valid metrics, including queue depth totals, per-channel breakdowns, and task status distribution. Validates end-to-end wiring from store through Observer to HTTP/UDS response.

---

### Preconditions
- [ ] AGH daemon running with task subsystem and observe layer initialized
- [ ] At least 1 task with 1 queued run exists (to produce non-zero queue metrics)

---

### Test Steps
1. **Query task summary projection**
   - Input: Request task summary from observe endpoint (HTTP or UDS)
   - **Expected:** 200 OK. Response includes:
     - `queue_depth_total`: integer >= 0
     - `queue_depth`: array of per-channel queue depth entries (may include unbound entry)
     - Task status distribution counters

2. **Verify queue depth reflects active runs**
   - Input: Create a task, enqueue a run (leaving it queued), then query summary
   - **Expected:** `queue_depth_total` >= 1. Queue depth array includes an entry for the unbound channel (or the run's channel) with count >= 1.

3. **Query task metrics**
   - Input: Request task metrics from observe endpoint
   - **Expected:** 200 OK. Response includes `TaskMetrics` structure with:
     - `task_queue_depth`: per-channel breakdown
     - Counts for tasks by status

4. **Verify metrics update after state change**
   - Input: Claim the queued run, then re-query metrics
   - **Expected:** `queue_depth_total` decremented by 1. Metrics reflect the new state.

5. **Verify metrics with filter parameter**
   - Input: Query metrics with `network_channel` filter
   - **Expected:** Response scoped to the specified channel. No data from other channels.

---

### Related Test Cases
- SMOKE-006: Enqueue and claim a run
- TC-PERF-006: Observe projection query performance
- SMOKE-001: Daemon starts with task subsystem
