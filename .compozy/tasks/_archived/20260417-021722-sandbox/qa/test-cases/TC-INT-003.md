## TC-INT-003: Session resume with local provider

**Priority:** P0 (Critical)
**Type:** Integration
**Status:** Not Run
**Estimated Time:** 2 minutes
**Created:** 2026-04-16
**Task:** 04

---

### Objective

Verify that session resume correctly restores sandbox metadata and passes it to the local provider's Prepare method, then the session works normally.

---

### Test Steps

1. **Create session, stop it**
   - **Expected:** Session stopped, metadata persisted

2. **Resume session**
   - **Expected:** Environment metadata restored, local provider Prepare called with prior SandboxID (no-op for local), session runs normally

3. **Verify resumed session can prompt and file IO**
   - **Expected:** ACP communication works, file operations work
