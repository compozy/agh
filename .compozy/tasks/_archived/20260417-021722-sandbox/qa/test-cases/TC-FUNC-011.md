## TC-FUNC-011: Session start calls SyncToRuntime after Prepare

**Priority:** P0 (Critical)
**Type:** Functional
**Status:** Not Run
**Estimated Time:** 1 minute
**Created:** 2026-04-16
**Task:** 04

---

### Objective

Verify that `startSession()` calls `Provider.SyncToRuntime(state, SyncReasonStart)` after a successful `Prepare()` and before `Launch`.

---

### Preconditions

- [x] Mock provider tracks call order

---

### Test Steps

1. **Create session and verify call order**
   - **Expected:** Call sequence is: `Prepare` -> `SyncToRuntime(SyncReasonStart)` -> `Launch`

2. **Verify SyncToRuntime receives correct SessionState**
   - **Expected:** `SessionState` includes `EnvironmentID`, `InstanceID`, `RuntimeRootDir` from Prepare result

3. **Verify SyncToRuntime skipped for local provider**
   - Input: Local provider (sync mode = none)
   - **Expected:** `SyncToRuntime` returns nil immediately (no-op)
