## TC-FUNC-017: SSH token refresh at 50% expiry

**Priority:** P1 (High)
**Type:** Functional
**Status:** Not Run
**Estimated Time:** 2 minutes
**Created:** 2026-04-16
**Task:** 06

---

### Objective

Verify that the SSH token manager proactively refreshes the token at 50% of the expiry window (e.g., at 30 minutes for a 60-minute token).

---

### Preconditions

- [x] SSH token manager with configurable expiry
- [x] Mock REST API for token fetch

---

### Test Steps

1. **Create token with 60-minute expiry**
   - **Expected:** Token stored with `SSHAccessExpiresAt` set to now + 60 minutes

2. **Check token at 29 minutes (before 50% threshold)**
   - **Expected:** No refresh triggered, existing token returned

3. **Check token at 31 minutes (past 50% threshold)**
   - **Expected:** Proactive refresh triggered, new token fetched, new expiry persisted

4. **Verify persisted expiry updated**
   - **Expected:** `SessionSandboxMeta.SSHAccessExpiresAt` updated to new expiry
