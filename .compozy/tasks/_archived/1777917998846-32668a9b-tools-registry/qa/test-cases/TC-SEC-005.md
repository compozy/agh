# TC-SEC-005 — Remote MCP `Authorization` header never crosses `internal/tools` boundary

- **Priority:** P0
- **Type:** Security / credential boundary
- **Trace:** Task 09, ADR-010, Safety Invariants 12, 20

## Objective

Prove the `Authorization` header injected by `internal/mcp` for an authenticated remote MCP call never appears in `internal/tools` errors, registry events, registry result envelopes, CLI/HTTP/UDS payloads, MCP responses surfaced back to AGH callers, or any log line.

## Preconditions

- Fake remote HTTP MCP server requires `Authorization: Bearer mcp:test:bearer:OAUTHTOKEN_v1`.
- Fake OAuth issuer returns `mcp:test:bearer:OAUTHTOKEN_v1` and `mcp:test:refresh:REFRESHTOKEN_v1`.
- Daemon configured with that MCP server in `[mcp_servers.fake_http]` (transport `http`).
- `agh mcp auth login fake_http` completed against fake issuer.

## Test Steps

1. Confirm `internal/mcp/auth` token store contains the bearer (test-only inspection of the store).
2. Invoke an MCP tool from `fake_http` through `agh tool invoke mcp__fake_http__echo`.
   - **Expected:** Call succeeds.
3. Capture daemon logs, CLI JSON output, HTTP API response body, UDS API response body, hosted MCP response (if exposed), and event journal.
4. Run sentinel scan from `security-redaction-regression.md` against `qa/logs/`.
   - **Expected:** No occurrence of `mcp:test:bearer:` or `mcp:test:refresh:` in any captured artifact.
5. Force the fake server to return 401 once.
   - **Expected:** Executor refreshes (TC-SEC-006); error path returned to caller carries only `mcp_auth_required`/`mcp_auth_invalid` redacted code, no header value.

## Edge Cases

- Verify SSE event payloads emitted by `internal/observe` redact the header value as well.
- Verify `agh tool info mcp__fake_http__echo -o json` shows redacted `auth_status` mirroring `/api/settings/mcp-servers` and never the bearer.

## Automation

- **Target:** Integration
- **Status:** Existing for redaction unit tests; Missing for integration end-to-end sentinel scan
- **Command/Spec:** `go test ./internal/mcp -run TestRemoteHTTPAuthRedaction`
- **Notes:** Critical because a leak here exposes user OAuth credentials.
