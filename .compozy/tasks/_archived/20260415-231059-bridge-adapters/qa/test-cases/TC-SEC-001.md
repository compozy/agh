## TC-SEC-001: HMAC-SHA256 Webhook Signature Verification

**Priority:** P0
**Type:** Security
**Risk Level:** Critical
**Status:** Not Run
**Estimated Time:** 30 minutes
**Created:** 2026-04-15

---

### Objective

Verify that HMAC-SHA256 webhook signature verification correctly authenticates requests for Slack, GitHub, and Linear providers, rejecting forged or missing signatures and preventing unauthorized webhook delivery.

### Preconditions

- [ ] Bridge adapter runtime is running with Slack, GitHub, and Linear providers registered
- [ ] Each provider instance is configured with a known `signing_secret`
- [ ] Webhook endpoints are accessible (e.g., `/webhooks/slack/{instance_id}`, `/webhooks/github/{instance_id}`, `/webhooks/linear/{instance_id}`)
- [ ] HTTP client capable of crafting custom headers and computing HMAC-SHA256 signatures is available (e.g., curl + openssl, or a test harness)

### Test Steps

1. **Valid signature — Slack provider**
   - Input: POST to Slack webhook endpoint with a valid JSON body. Compute `X-Slack-Signature` using `v0:timestamp:body` format with the correct signing secret. Include `X-Slack-Request-Timestamp` header.
   - **Expected:** Request accepted (200 OK), event delivered to the bridge instance.

2. **Valid signature — GitHub provider**
   - Input: POST to GitHub webhook endpoint with a valid JSON body. Compute `X-Hub-Signature-256` as `sha256=<hmac>` using the correct webhook secret.
   - **Expected:** Request accepted (200 OK), event delivered to the bridge instance.

3. **Valid signature — Linear provider**
   - Input: POST to Linear webhook endpoint with a valid JSON body. Compute the Linear signature header using the correct signing secret.
   - **Expected:** Request accepted (200 OK), event delivered to the bridge instance.

4. **Invalid signature — wrong secret**
   - Input: For each provider (Slack, GitHub, Linear), compute the HMAC-SHA256 signature using an incorrect secret (e.g., `wrong-secret-value`). Send the request with this forged signature.
   - **Expected:** Request rejected with 401 or 403. No event delivered. Response body does not leak the expected signature or secret.

5. **Invalid signature — tampered body**
   - Input: For each provider, compute a valid signature for body `{"event":"original"}`, then send the request with a modified body `{"event":"tampered"}` but the original signature.
   - **Expected:** Request rejected with 401 or 403. Signature mismatch detected. No event delivered.

6. **Missing signature header**
   - Input: For each provider, send a valid POST request with correct Content-Type and body but omit the signature header entirely (`X-Slack-Signature`, `X-Hub-Signature-256`, or Linear equivalent).
   - **Expected:** Request rejected with 401 or 403. Error message indicates missing signature, not an internal server error.

7. **Empty signature header**
   - Input: For each provider, send the request with the signature header present but set to an empty string.
   - **Expected:** Request rejected with 401 or 403. No crash or panic from empty string comparison.

8. **Replay attack — stale timestamp (Slack)**
   - Input: Send a Slack webhook request with a valid signature but `X-Slack-Request-Timestamp` set to more than 5 minutes in the past.
   - **Expected:** Request rejected. Timestamp staleness check prevents replay attacks.

9. **Timing-safe comparison verification**
   - Input: Send two requests with invalid signatures: one that shares the first 16 bytes with the valid signature, and one that differs entirely. Measure response times for both.
   - **Expected:** Response times are statistically indistinguishable (within noise), confirming `hmac.Equal()` or `crypto/subtle.ConstantTimeCompare()` is used rather than byte-by-byte comparison.

10. **Signature with incorrect encoding**
    - Input: Send a request with the signature encoded in base64 instead of hex (or vice versa, depending on expected format).
    - **Expected:** Request rejected with 401 or 403. No panic from decoding errors.

### Attack Vectors

- [ ] Signature forgery with guessed or leaked secret
- [ ] Body tampering after signature computation
- [ ] Replay attacks using captured valid requests with old timestamps
- [ ] Timing side-channel attacks on signature comparison
- [ ] Missing or malformed signature headers causing unexpected code paths
- [ ] Encoding confusion (hex vs base64) leading to bypass

### Related Test Cases

- TC-SEC-002 (Ed25519 verification for Discord)
- TC-SEC-003 (Method validation — ensures POST-only before signature check)
- TC-SEC-004 (Body size limits — ensures oversized bodies rejected before signature verification)
