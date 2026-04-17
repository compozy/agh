## TC-FUNC-015: Session resume restores environment metadata

**Priority:** P0 (Critical)
**Type:** Functional
**Status:** Not Run
**Estimated Time:** 1 minute
**Created:** 2026-04-16
**Task:** 04

---

### Objective

Verify that on session resume, `SessionEnvironmentMeta` is correctly restored and passed to `Provider.Prepare()` with `EnvironmentID`, `InstanceID`, and `ProviderState` to enable sandbox reattachment.

---

### Preconditions

- [x] Session previously created with environment metadata persisted
- [x] Session in resumable state

---

### Test Steps

1. **Create session, stop it, then resume**
   - **Expected:** `PrepareRequest` on resume includes `EnvironmentID` from original session, `InstanceID` from prior Prepare, `ProviderState` from prior persist

2. **Verify provider reattaches (not creates new)**
   - **Expected:** Mock provider receives non-empty `InstanceID`, indicating reattach rather than fresh create

3. **Verify environment state transitions**
   - **Expected:** State goes from stopped -> creating -> prepared -> running
