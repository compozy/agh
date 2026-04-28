## TC-PERF-003: Sandbox reconciliation boot time

**Priority:** P2 (Medium)
**Type:** Performance
**Status:** Not Run
**Estimated Time:** 1 minute
**Created:** 2026-04-16
**Task:** 07

---

### Objective

Verify that sandbox reconciliation during daemon boot does not add significant latency, especially when there are no remote sessions to reconcile.

---

### Performance Criteria

| Metric | Target | Acceptable | Actual | Status |
|--------|--------|------------|--------|--------|
| No remote sessions | < 1ms | < 5ms | | [ ] |
| 10 local-only sessions | < 5ms | < 10ms | | [ ] |
| 5 remote sessions (mock) | < 100ms | < 500ms | | [ ] |

---

### Test Steps

1. **Boot with no sessions**
   - **Expected:** Reconciliation adds < 1ms

2. **Boot with 10 local sessions**
   - **Expected:** Quick scan, all skipped (local backend), < 5ms

3. **Boot with remote sessions (mocked provider)**
   - **Expected:** Each reattach attempt within timeout, total < 500ms for 5 sessions
