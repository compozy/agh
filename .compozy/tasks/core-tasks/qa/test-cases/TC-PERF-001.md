## TC-PERF-001: Sequential Task Creation Throughput (1000 Tasks)

**Priority:** P1
**Type:** Performance
**Status:** Not Run
**Created:** 2026-04-14

---

### Objective
Validate that the task system can create 1000 tasks sequentially within acceptable latency bounds. This measures the core write path through validation, identity derivation, store persistence, audit event emission, and dependency status reconciliation.

---

### Preconditions
- [ ] AGH daemon running with task subsystem initialized
- [ ] Clean task store (no pre-existing tasks to avoid index contention noise)
- [ ] Authenticated human principal via HTTP or UDS ingress
- [ ] System under normal load (no concurrent heavy operations)

---

### Performance Criteria
| Metric | Target | Acceptable | Actual | Status |
|--------|--------|------------|--------|--------|
| Total wall-clock time for 1000 creates | <500ms | <1000ms | | [ ] |
| Average latency per task creation | <0.5ms | <1ms | | [ ] |
| P99 latency per task creation | <2ms | <5ms | | [ ] |
| Memory delta during creation burst | <50MB | <100MB | | [ ] |
| Zero validation or persistence errors | 0 errors | 0 errors | | [ ] |

---

### Test Steps
1. **Seed authenticated actor context**
   - Derive a human actor context with `FullAccessAuthority()` for CLI origin
   - **Expected:** Actor context valid, no error

2. **Create 1000 global-scoped tasks sequentially**
   - Input: Loop 1000 iterations, each calling `CreateTask` with unique title `"perf-task-NNN"` and scope `"global"`
   - Record wall-clock start time before loop, end time after loop
   - **Expected:** All 1000 tasks created successfully. Total duration < 500ms.

3. **Verify all tasks persisted**
   - Input: `ListTasks` with no filters, limit 1001
   - **Expected:** Exactly 1000 task summaries returned. All have status `"pending"` or `"ready"`.

4. **Measure individual operation latency**
   - Input: Record per-iteration latency during step 2
   - **Expected:** P50 < 0.3ms, P99 < 2ms. No outlier > 10ms.

5. **Repeat with workspace-scoped tasks**
   - Input: Create 1000 workspace-scoped tasks with workspace ID `"perf-ws"`
   - **Expected:** Similar throughput. Workspace binding does not introduce significant overhead.

---

### Related Test Cases
- TC-PERF-003: Hierarchy depth and child count limits
- TC-PERF-005: ListTasks filter performance on large dataset
- SMOKE-002: Basic task creation
