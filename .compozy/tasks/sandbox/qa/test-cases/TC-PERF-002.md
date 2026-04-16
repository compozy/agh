## TC-PERF-002: Session start latency with local provider

**Priority:** P2 (Medium)
**Type:** Performance
**Status:** Not Run
**Estimated Time:** 2 minutes
**Created:** 2026-04-16
**Task:** 04

---

### Objective

Verify that the environment abstraction layer adds negligible overhead to local session start time compared to pre-extraction baseline.

---

### Performance Criteria

| Metric | Target | Acceptable | Actual | Status |
|--------|--------|------------|--------|--------|
| Local Prepare overhead | < 1ms | < 5ms | | [ ] |
| Local SyncToRuntime overhead | < 1ms | < 5ms | | [ ] |
| Total environment lifecycle overhead | < 5ms | < 10ms | | [ ] |

---

### Test Steps

1. **Measure local provider Prepare duration**
   - **Expected:** Essentially zero overhead (returns immediately)

2. **Measure total session start time**
   - **Expected:** Environment lifecycle adds < 5ms to session start

3. **Compare with pre-extraction baseline if available**
   - **Expected:** No measurable regression
