## TC-FUNC-016: Session crash calls SyncFromRuntime best-effort

**Priority:** P0 (Critical)
**Type:** Functional
**Status:** Not Run
**Estimated Time:** 1 minute
**Created:** 2026-04-16
**Task:** 04

---

### Objective

Verify that on session crash, `SyncFromRuntime(state, SyncReasonCrash)` is called as best-effort. If sync fails, the error is logged but does not prevent session cleanup.

---

### Preconditions

- [x] Mock provider that can simulate sync failure

---

### Test Steps

1. **Simulate session crash and verify sync attempt**
   - **Expected:** `SyncFromRuntime` called with `SyncReasonCrash`

2. **Simulate sync failure during crash path**
   - Input: Provider returns error from SyncFromRuntime
   - **Expected:** Error logged, session cleanup continues, no panic or hang

3. **Verify Destroy still called after failed sync**
   - **Expected:** `Destroy()` called even if SyncFromRuntime errored (for transient persistence)
