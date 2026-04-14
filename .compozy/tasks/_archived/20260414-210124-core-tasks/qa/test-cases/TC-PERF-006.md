## TC-PERF-006: Observe Projection Queries on 10K Tasks + 50K Runs

**Priority:** P2
**Type:** Performance
**Status:** Not Run
**Created:** 2026-04-14

---

### Objective
Validate that the observe layer's task projection queries (queue depth, stuck work detection, task metrics) return results within 500ms when operating on a store containing 10,000 tasks and 50,000 runs. This measures the efficiency of the `Observer.QueryTaskSummary` and `Observer.QueryTaskMetrics` aggregation paths.

---

### Preconditions
- [ ] AGH daemon running with task subsystem and observe layer initialized
- [ ] 10,000 tasks seeded across global and workspace scopes
- [ ] 50,000 task runs seeded with varied statuses:
  - 10,000 queued, 10,000 claimed, 5,000 starting, 10,000 running, 10,000 completed, 3,000 failed, 2,000 cancelled
- [ ] Runs distributed across 5 network channels and unbound runs
- [ ] Task events seeded for audit trail queries (estimated 100K+ events)
- [ ] SQLite WAL mode enabled

---

### Performance Criteria
| Metric | Target | Acceptable | Actual | Status |
|--------|--------|------------|--------|--------|
| QueryTaskSummary (queue depth) | <500ms | <1000ms | | [ ] |
| QueryTaskMetrics (unfiltered) | <500ms | <1000ms | | [ ] |
| QueryTaskMetrics (filtered by origin_kind) | <500ms | <1000ms | | [ ] |
| QueryTaskMetrics (filtered by network_channel) | <300ms | <500ms | | [ ] |
| Queue depth by channel breakdown | <200ms | <500ms | | [ ] |
| Stuck work detection (runs > threshold age) | <500ms | <1000ms | | [ ] |

---

### Test Steps
1. **Seed 10K tasks and 50K runs**
   - Input: Programmatically create tasks with 5 runs per task on average, distributed across statuses and channels
   - **Expected:** All data persisted. Seeding completes within 2 minutes.

2. **Query task summary projection**
   - Input: `Observer.QueryTaskSummary(ctx)` -- full snapshot
   - Record response time
   - **Expected:** Response in < 500ms. `QueueDepth` array populated with per-channel counts. `QueueDepthTotal` matches sum of queued runs.

3. **Query task metrics (unfiltered)**
   - Input: `Observer.QueryTaskMetrics(ctx, TaskMetricsQuery{})`
   - **Expected:** Response in < 500ms. `TaskQueueDepth` shows per-channel breakdown. Counters match expected distributions.

4. **Query task metrics filtered by origin kind**
   - Input: `Observer.QueryTaskMetrics(ctx, TaskMetricsQuery{OriginKind: "cli"})`
   - **Expected:** Response in < 500ms. Only CLI-originated metrics included.

5. **Query task metrics filtered by network channel**
   - Input: `Observer.QueryTaskMetrics(ctx, TaskMetricsQuery{NetworkChannel: "builders"})`
   - **Expected:** Response in < 300ms. Channel filter narrows the dataset significantly.

6. **Stuck work detection query**
   - Input: Seed 100 runs with `claimed_at` > 10 minutes ago but still in `claimed` status (stuck). Query for stuck work.
   - **Expected:** Detection completes in < 500ms. All 100 stuck runs identified.

7. **Concurrent projection queries**
   - Input: 5 concurrent `QueryTaskMetrics` with different filters
   - **Expected:** All complete within < 1000ms. SQLite WAL handles concurrent reads without blocking.

---

### Related Test Cases
- TC-PERF-005: ListTasks filter performance
- SMOKE-010: Observe projections return task metrics
