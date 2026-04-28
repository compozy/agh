## TC-FUNC-008: Session start allocates SandboxID

**Priority:** P0 (Critical)
**Type:** Functional
**Status:** Not Run
**Estimated Time:** 1 minute
**Created:** 2026-04-16
**Task:** 04

---

### Objective

Verify that `startSession()` allocates a daemon-owned `SandboxID` before any provider calls, and this ID is unique per session.

---

### Preconditions

- [x] Session manager has sandbox registry injected
- [x] `SessionSandboxMeta` type available

---

### Test Steps

1. **Create a session**
   - **Expected:** `SessionSandboxMeta.SandboxID` is non-empty before `Provider.Prepare()` is called

2. **Create two sessions**
   - **Expected:** Each session has a distinct `SandboxID`

3. **Verify SandboxID format**
   - **Expected:** UUID or similar unique identifier format
