## TC-FUNC-009: Session start persists meta in creating state

**Priority:** P0 (Critical)
**Type:** Functional
**Status:** Not Run
**Estimated Time:** 1 minute
**Created:** 2026-04-16
**Task:** 04

---

### Objective

Verify that `startSession()` persists `SessionEnvironmentMeta` with `State = "creating"` before calling `Provider.Prepare()`. This ensures recovery is possible if Prepare fails or times out.

---

### Preconditions

- [x] Session environment columns exist in sessions table
- [x] Provider.Prepare can be intercepted/mocked

---

### Test Steps

1. **Create session and inspect metadata before Prepare completes**
   - **Expected:** `SessionEnvironmentMeta.State == "creating"`, `EnvironmentID` non-empty, `Backend` matches resolved profile

2. **Simulate Prepare failure**
   - Input: Mock provider that returns error from Prepare
   - **Expected:** Session metadata still has `creating` state persisted, session fails with error but metadata is recoverable
