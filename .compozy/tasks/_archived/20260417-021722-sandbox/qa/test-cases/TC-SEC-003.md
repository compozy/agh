## TC-SEC-003: Env var allowlist allows AGH_SESSION_ID

**Priority:** P0 (Critical)
**Type:** Security
**Status:** Not Run
**Estimated Time:** 1 minute
**Created:** 2026-04-16
**Task:** 06
**Risk Level:** Medium

---

### Objective

Verify that `AGH_*` session-specific environment variables are correctly propagated to remote sandboxes through the allowlist.

---

### Test Steps

1. **Verify AGH_SESSION_ID propagated**
   - Input: `AGH_SESSION_ID=sess-123`
   - **Expected:** Present in sandbox environment

2. **Verify AGH_SESSION_CHANNEL propagated**
   - Input: `AGH_SESSION_CHANNEL=chan-456`
   - **Expected:** Present in sandbox environment

3. **Verify AGH_PEER_ID propagated**
   - Input: `AGH_PEER_ID=peer-789`
   - **Expected:** Present in sandbox environment

4. **Verify profile env overrides merged**
   - Input: Profile `Env = {"CUSTOM_VAR": "value"}`
   - **Expected:** `CUSTOM_VAR=value` present in sandbox environment
