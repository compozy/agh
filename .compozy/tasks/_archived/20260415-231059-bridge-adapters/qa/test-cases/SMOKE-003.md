## SMOKE-003: Webhook Signature Verification Accepts Valid Request

**Priority:** P0
**Type:** Smoke
**Status:** Not Run
**Estimated Time:** 2 minutes
**Created:** 2026-04-15

---

### Objective

Verify that at least one provider (Slack) accepts a webhook request with a valid HMAC-SHA256 signature and rejects one with an invalid signature.

### Preconditions

- [ ] Slack provider extension compiles
- [ ] Test signing secret available

### Test Steps

1. **Compute valid HMAC-SHA256 signature for a test payload using the signing secret**
   - **Expected:** Signature computed successfully

2. **Send POST request to webhook endpoint with valid signature header**
   - **Expected:** Request accepted (200 or 202), no signature error

3. **Send POST request with tampered signature**
   - **Expected:** Request rejected with 401/403

### Related Test Cases

- TC-SEC-001, TC-SEC-002
