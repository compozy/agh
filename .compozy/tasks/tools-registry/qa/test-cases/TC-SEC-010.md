# TC-SEC-010 — Hosted MCP bind nonce is single-use, TTL-bounded, and redacted

- **Priority:** P0
- **Type:** Security / hosted MCP lifecycle
- **Trace:** Task 02 (`bind_nonce_ttl_seconds`), Task 10, ADR-002, Safety Invariant 16

## Objective

Prove the hosted MCP launch record is invalidated on first successful bind, on session end, on proxy disconnect, or on TTL expiry — whichever happens first — and the nonce is redacted from logs/diagnostics.

## Preconditions

- `[tools.hosted_mcp].bind_nonce_ttl_seconds = 5` (lower bound is 5).
- Test harness can advance daemon clock or wait real time.

## Test Steps

1. Mint a nonce; bind successfully.
   - **Expected:** Bind succeeds; daemon log contains a redacted correlation id (e.g. `bind_nonce=<redacted:correlation>`), never the raw value.
2. Attempt second bind with the same nonce.
   - **Expected:** Rejected — single-use.
3. Mint a new nonce, wait > 5 seconds without binding.
   - **Expected:** Bind rejected after TTL expiry.
4. Restart daemon; pending nonce invalidated.
   - **Expected:** Subsequent bind rejected; session must mint fresh nonce.
5. Bind successfully then terminate proxy.
   - **Expected:** Launch record invalidated immediately; new bind requires fresh nonce.
6. Sentinel scan for `BIND_NONCE_v1_TESTONLY` across all logs and diagnostics.
   - **Expected:** No match; only redacted correlation references appear.

## Edge Cases

- TTL bounds `[5, 300]` validated by config (TC-FUNC-014).
- Session/load issues a fresh nonce per resume; no reuse across resumes.

## Automation

- **Target:** Integration
- **Status:** Existing partial; Missing redaction sentinel scan integration
- **Command/Spec:** `go test ./internal/mcp -run TestHostedMCPNonceLifecycle`
