## TC-SEC-007: Secret Binding Isolation

**Priority:** P1
**Type:** Security
**Risk Level:** High
**Status:** Not Run
**Estimated Time:** 30 minutes
**Created:** 2026-04-15

---

### Objective
Verify that bound secrets (bot tokens, signing secrets, API keys, webhook secrets) are securely isolated per instance, resolved only at initialization, never exposed in API responses, never written to logs or marker files, and inaccessible to other provider instances.

### Preconditions
- [ ] Bridge adapter runtime is running with at least two provider instances:
  - Instance A (Slack) with bound secrets: `bot_token`, `signing_secret`
  - Instance B (Discord) with bound secrets: `bot_token`, `public_key`
- [ ] Log output is captured and inspectable (stdout, file, or structured log sink)
- [ ] Host API endpoints are accessible for instance queries
- [ ] Filesystem access to the runtime's working directory (for marker file inspection)
- [ ] Ability to trigger secret resolution (e.g., restart an instance or create a new one)

### Test Steps

1. **Secrets resolved at initialization only**
   - Input: Create a new provider instance with `bot_token: "secret-token-abc123"`. Monitor the initialization sequence.
   - **Expected:** Secret is resolved (fetched from secret store or config) during instance initialization. No subsequent re-resolution on each webhook request. In-memory cache holds the resolved value.

2. **GET instance response omits secrets**
   - Input: Call `instances/get` for Instance A via the Host API.
   - **Expected:** Response includes instance metadata (ID, provider type, status) but does NOT include `bot_token`, `signing_secret`, or any other secret values. Secret fields are either absent from the response or redacted (e.g., `"bot_token": "***"`).

3. **LIST instances response omits secrets**
   - Input: Call `instances/list` via the Host API.
   - **Expected:** No instance in the list response includes secret values. Secrets are consistently omitted across all API response types.

4. **Secrets not in log output — initialization**
   - Input: Set log level to DEBUG. Create a new instance with `bot_token: "xoxb-super-secret-value"`. Capture all log output during initialization.
   - **Expected:** The string `xoxb-super-secret-value` does not appear anywhere in the log output. Logs may reference that a secret was resolved (e.g., "bot_token resolved successfully") but never log the actual value.

5. **Secrets not in log output — webhook processing**
   - Input: Send a webhook request to Instance A. Capture all log output during request processing, including error cases (e.g., invalid signature).
   - **Expected:** Neither `bot_token` nor `signing_secret` values appear in any log line. Signature verification failures log the event but not the expected or actual signature values.

6. **Secrets not in log output — error paths**
   - Input: Trigger an error condition that involves secrets (e.g., use an invalid bot_token that fails API calls, or misconfigure the signing_secret). Capture error logs.
   - **Expected:** Error messages describe the failure (e.g., "authentication failed", "invalid token") without including the secret value itself.

7. **Secrets not written to marker files**
   - Input: Inspect all files in the runtime's working directory and data directory after instance creation and webhook processing.
   - **Expected:** No file on disk contains secret values in plaintext. Marker files (if any) contain instance IDs or status but not secrets.

8. **Secret isolation between instances**
   - Input: Instance A (Slack) has `signing_secret: "slack-secret-123"`. Instance B (Discord) has `public_key: "discord-key-456"`. Send a webhook to Instance B.
   - **Expected:** Instance B's signature verification uses only `discord-key-456`. There is no code path where Instance A's `slack-secret-123` could be accessed by Instance B's processing logic.

9. **Secret not returned in error responses**
   - Input: Send a webhook with an invalid signature to Instance A. Inspect the HTTP error response body.
   - **Expected:** Response body contains a generic error message (e.g., `{"error":"signature verification failed"}`). No secret material in the response. No stack trace exposing in-memory secret values.

10. **Secrets not accessible via environment variable leak**
    - Input: If secrets are sourced from environment variables, verify that the runtime does not expose environment variables through any API endpoint (e.g., debug, health, status endpoints).
    - **Expected:** No API endpoint returns environment variable values. Health/status endpoints return only operational metrics, not configuration or secrets.

11. **Secret rotation — old secret invalidated**
    - Input: Update Instance A's `signing_secret` to a new value (if hot-reconfiguration is supported). Send a webhook signed with the old secret.
    - **Expected:** Request rejected. The old secret is no longer valid. Only the new secret is accepted for signature verification.

12. **Memory inspection resistance (best effort)**
    - Input: After instance initialization, trigger a heap dump or memory profile (if available in test environment).
    - **Expected:** Secrets are stored in memory (necessary for operation) but are not duplicated unnecessarily across multiple data structures. This is a best-effort verification.

### Attack Vectors
- [ ] API response scraping to extract secrets from GET/LIST endpoints
- [ ] Log harvesting to find secrets in plaintext log output
- [ ] Marker file inspection to find secrets written to disk
- [ ] Cross-instance secret leakage via shared data structures or global state
- [ ] Error response analysis to extract secrets from verbose error messages
- [ ] Environment variable exposure through debug or diagnostic endpoints
- [ ] Memory dump analysis to find secrets in process memory
- [ ] Secret persistence after rotation (old secrets remaining valid)

### Related Test Cases
- TC-SEC-001 (Signature verification — uses the bound signing secret)
- TC-SEC-002 (Ed25519 verification — uses the bound public key)
- TC-SEC-006 (Instance ownership — complementary isolation at the access control level)
- TC-SEC-010 (Config injection — prevents secrets from being injected via config fields)
