## TC-FUNC-022: Reconciliation destroys unrecoverable session

**Priority:** P1 (High)
**Type:** Functional
**Status:** Not Run
**Estimated Time:** 1 minute
**Created:** 2026-04-16
**Task:** 07

---

### Objective

Verify that reconciliation calls `Provider.Destroy()` for unrecoverable remote sessions and logs the cleanup action.

---

### Preconditions

- [x] Session with remote backend that provider cannot reattach
- [x] Mock provider that returns error on Prepare

---

### Test Steps

1. **Boot with unrecoverable remote session**
   - Input: Provider.Prepare returns "sandbox not found" error
   - **Expected:** `Destroy()` called for cleanup, structured log entry records the cleanup action with session ID and sandbox ID

2. **Verify boot continues after cleanup**
   - **Expected:** Daemon boot completes successfully despite the failed reattach
