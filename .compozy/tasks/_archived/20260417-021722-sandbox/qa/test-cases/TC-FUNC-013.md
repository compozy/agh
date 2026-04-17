## TC-FUNC-013: Session stop calls SyncFromRuntime

**Priority:** P0 (Critical)
**Type:** Functional
**Status:** Not Run
**Estimated Time:** 1 minute
**Created:** 2026-04-16
**Task:** 04

---

### Objective

Verify that on graceful session stop, `finalizeStopped` calls `Provider.SyncFromRuntime(state, SyncReasonStop)` before store close.

---

### Preconditions

- [x] Mock provider tracks SyncFromRuntime calls

---

### Test Steps

1. **Create session, then stop it**
   - **Expected:** `SyncFromRuntime` called with `SyncReasonStop`, `SessionState` includes current environment metadata

2. **Verify call order**
   - **Expected:** `SyncFromRuntime` -> store close -> `Destroy` (if applicable)

3. **Verify sync result logged**
   - **Expected:** Structured log includes `duration_ms`, `files_synced`, `bytes_transferred`
