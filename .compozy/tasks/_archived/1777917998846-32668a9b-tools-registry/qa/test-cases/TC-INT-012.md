# TC-INT-012 — Remote OAuth-protected MCP call-through with token redaction

- **Priority:** P1
- **Type:** Integration / MCP / security
- **Trace:** Task 09, ADR-010, Safety Invariant 20

## Test Steps

1. Configure remote HTTP MCP server `fake_http` with `MCPAuthConfig` pointing to fake OAuth issuer.
2. `agh mcp auth login fake_http` against the fake issuer.
3. Invoke a tool through `Registry.Call`.
   - **Expected:** Call succeeds; outbound `Authorization: Bearer ...` injected only inside `internal/mcp`.
4. Capture all logs/events/payloads; sentinel scan.
   - **Expected:** No leak of `mcp:test:bearer:...` (covered by TC-SEC-005).
5. `agh mcp auth status fake_http -o json` reports `authenticated`.
6. Force expiry; observe registry availability switches to `mcp_auth_expired`.

## Automation

- **Target:** Integration
- **Status:** Existing
- **Command/Spec:** `go test ./internal/mcp -run TestRemoteOAuthCallThrough`
