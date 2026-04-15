## TC-SEC-002: Ed25519 Webhook Signature Verification

**Priority:** P0
**Type:** Security
**Risk Level:** Critical
**Status:** Not Run
**Estimated Time:** 25 minutes
**Created:** 2026-04-15

---

### Objective

Verify that Ed25519 webhook signature verification for the Discord provider correctly validates request authenticity, rejecting tampered bodies, forged signatures, and missing headers.

### Preconditions

- [ ] Bridge adapter runtime is running with a Discord provider instance registered
- [ ] Discord instance is configured with a known Ed25519 public key (from the Discord application settings)
- [ ] Corresponding Ed25519 private key is available in the test harness for generating valid test signatures
- [ ] Webhook endpoint is accessible (e.g., `/webhooks/discord/{instance_id}`)
- [ ] HTTP client capable of crafting custom headers is available

### Test Steps

1. **Valid Ed25519 signature**
   - Input: POST to Discord webhook endpoint with a valid JSON body (e.g., Discord interaction payload). Sign the concatenation of `X-Signature-Timestamp` + body using the Ed25519 private key. Set `X-Signature-Ed25519` to the hex-encoded signature and `X-Signature-Timestamp` to the current Unix timestamp.
   - **Expected:** Request accepted (200 OK). Event delivered to the bridge instance.

2. **Invalid signature — wrong key pair**
   - Input: Generate a different Ed25519 key pair. Sign the same timestamp + body with the wrong private key. Send with the forged `X-Signature-Ed25519` header.
   - **Expected:** Request rejected with 401 or 403. No event delivered. Response does not expose the expected public key.

3. **Tampered body with original signature**
   - Input: Generate a valid signature for body `{"type":1}`. Send the request with body `{"type":1,"injected":"malicious"}` but keep the original signature.
   - **Expected:** Request rejected with 401 or 403. Ed25519 verification detects body modification.

4. **Tampered timestamp with original signature**
   - Input: Generate a valid signature for timestamp `1700000000` + body. Send the request with `X-Signature-Timestamp: 1700000001` (off by one) but keep the original signature.
   - **Expected:** Request rejected. The timestamp is part of the signed message, so any change invalidates the signature.

5. **Missing `X-Signature-Ed25519` header**
   - Input: Send a valid POST with `X-Signature-Timestamp` present but omit `X-Signature-Ed25519`.
   - **Expected:** Request rejected with 401 or 403. Clear error indicating missing signature header.

6. **Missing `X-Signature-Timestamp` header**
   - Input: Send a valid POST with `X-Signature-Ed25519` present but omit `X-Signature-Timestamp`.
   - **Expected:** Request rejected with 401 or 403. Clear error indicating missing timestamp header.

7. **Both signature headers missing**
   - Input: Send a POST with correct Content-Type and body but no Discord signature headers at all.
   - **Expected:** Request rejected with 401 or 403. No panic or unhandled nil reference.

8. **Malformed signature — invalid hex encoding**
   - Input: Send `X-Signature-Ed25519: not-valid-hex-zzzz` with a valid timestamp.
   - **Expected:** Request rejected with 401 or 403. Hex decoding error handled gracefully, no 500 or panic.

9. **Truncated signature**
   - Input: Send `X-Signature-Ed25519` with only the first 32 bytes of a valid 64-byte Ed25519 signature (hex-encoded).
   - **Expected:** Request rejected. Length validation catches the short signature before verification attempt.

10. **Discord PING interaction with valid signature**
    - Input: Send a Discord `{"type":1}` PING interaction with a valid Ed25519 signature.
    - **Expected:** Request accepted. Response is `{"type":1}` (PONG). This is required by Discord's verification handshake.

### Attack Vectors

- [ ] Signature forgery using a different Ed25519 key pair
- [ ] Body injection after valid signature was computed
- [ ] Timestamp manipulation to alter the signed message
- [ ] Missing or partial headers causing nil dereference or unhandled errors
- [ ] Malformed hex encoding in signature header
- [ ] Truncated signatures bypassing length checks
- [ ] Replay of a valid signature with altered timestamp

### Related Test Cases

- TC-SEC-001 (HMAC-SHA256 verification for Slack, GitHub, Linear)
- TC-SEC-003 (Method validation — ensures POST-only before signature check)
- TC-SEC-004 (Body size limits — ensures oversized bodies rejected before signature verification)
