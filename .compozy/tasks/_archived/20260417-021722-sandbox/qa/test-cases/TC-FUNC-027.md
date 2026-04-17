## TC-FUNC-027: environment.sync.before deny skips sync

**Priority:** P2 (Medium)
**Type:** Functional
**Status:** Not Run
**Estimated Time:** 1 minute
**Created:** 2026-04-16
**Task:** 08

---

### Objective

Verify that when the `environment.sync.before` sync hook returns a `ControlPatch.Deny`, the sync operation is skipped (not errored).

---

### Preconditions

- [x] Hook registered for `environment.sync.before` that returns Deny

---

### Test Steps

1. **Stop session with hook that denies sync**
   - Input: Hook returns `{Deny: true, DenyReason: "skip sync per policy"}`
   - **Expected:** `SyncFromRuntime` is NOT called, session stop continues normally

2. **Verify environment.sync.after NOT fired**
   - **Expected:** Since sync was skipped, the after hook should not fire (or fire with zero stats)
