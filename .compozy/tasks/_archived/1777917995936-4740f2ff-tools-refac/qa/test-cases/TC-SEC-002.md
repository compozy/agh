# TC-SEC-002: MCP Auth Status Redaction

**Priority:** P0 (Critical)
**Type:** Security / Redaction
**Status:** Not Run
**Estimated Time:** 20 minutes
**Created:** 2026-04-30
**Last Updated:** 2026-04-30

## Objective

Prove `agh__mcp_auth_status` and the equivalent CLI/HTTP/UDS/settings surfaces never expose token / refresh-token / authorization code / PKCE verifier / callback secret material. Confirm `token_present` is the only public diagnostic and that no operator log line carries the secret either.

## Traceability

- Task: task_10.
- TechSpec: "Hosted MCP", "Existing MCP Config And Auth", "Test Strategy → Unit Tests".
- ADR: ADR-004.
- Surfaces: `internal/tools/builtin/mcp_auth.go`, `internal/tools/mcp.go`, `internal/mcp/auth/service.go`, `internal/api/core/conversions.go`, `internal/cli/mcp_auth.go`.

## Preconditions

- Isolated `AGH_HOME`.
- Mock OAuth server seeded with deterministic tokens, codes, PKCE verifiers, and client secrets. (Test fixtures only — no production credentials.)
- `mcp-server-a` connected; `mcp-server-b` token expired but stored in the auth state.

## Test Steps

1. **Tool invocation:**
   ```bash
   agh tool invoke agh__mcp_auth_status --input '{"server_name":"mcp-server-a"}' -o json \
     | tee qa/logs/TC-SEC-002/tool-status-a.json
   agh tool invoke agh__mcp_auth_status --input '{"server_name":"mcp-server-b"}' -o json \
     | tee qa/logs/TC-SEC-002/tool-status-b.json
   ```

2. **CLI:**
   ```bash
   agh mcp auth status --server mcp-server-a -o json | tee qa/logs/TC-SEC-002/cli-status-a.json
   agh mcp auth status --server mcp-server-b -o json | tee qa/logs/TC-SEC-002/cli-status-b.json
   ```

3. **HTTP/UDS settings payload:**
   ```bash
   curl -s --unix-socket "$AGH_HOME/run/sock/uds.sock" "http://localhost/api/settings/mcp" \
     | tee qa/logs/TC-SEC-002/settings.json
   ```

4. **Daemon log:**
   ```bash
   cat $AGH_HOME/logs/daemon.log | tee qa/logs/TC-SEC-002/daemon.log
   ```

5. **Cross-channel grep:**
   ```bash
   grep -RIn -E "access_token|refresh_token|client_secret|code=|pkce|callback" qa/logs/TC-SEC-002 \
     | tee qa/logs/TC-SEC-002/grep.txt
   ```
   - **Expected:** Zero matches.

6. Confirm only allowed public fields are present:
   ```bash
   jq -r 'keys[]' qa/logs/TC-SEC-002/tool-status-a.json | sort > qa/logs/TC-SEC-002/keys.txt
   ```
   - **Expected:** Keys are a subset of `{server_name, status, auth_type, client_id, scopes, expires_at, refreshable, token_present, diagnostic}`. No token-bearing field.

7. **Refresh-side-effect check:**
   - Configure mock OAuth to record any refresh request.
   - Invoke `agh__mcp_auth_status` ten times in a tight loop.
   - **Expected:** Mock server log shows zero refresh requests caused by these calls.

8. Run focused Go tests:
   ```bash
   go test ./internal/tools/builtin -run "TestMCPAuth" -count=1 | tee qa/logs/TC-SEC-002/builtin-tests.log
   go test ./internal/mcp/auth -count=1 | tee qa/logs/TC-SEC-002/auth-tests.log
   ```

## Evidence To Capture

- Tool / CLI / settings payloads.
- Daemon log filtered for `mcp` and `auth` prefixes.
- Mock OAuth server log under `qa/logs/TC-SEC-002/mock-oauth.log`.
- Cross-channel grep output and keys list.
- Test logs.

## Edge Cases And Variations

| Variation | Input | Expected Result |
|-----------|-------|-----------------|
| Auth type bearer with stored token | server with bearer credentials | `token_present=true`, no token value in any output |
| Auth type none | server without auth | Status reports `not_required`; no fields fabricated |
| Diagnostic message containing redirect URL | OAuth flow misconfigured | Diagnostic mentions repair via `agh mcp auth login` but does not embed the URL or code parameters |
| Settings UI payload | `/api/settings/mcp` consumer | Only redacted fields surface; `auth_status` key present, secret keys absent |

## Channels Exercised

- Tool invoke, CLI, HTTP/UDS settings, daemon log, mock OAuth.

## Related Test Cases

- TC-FUNC-008 (MCP auth status diagnostics).
- TC-INT-003 (hosted MCP projection parity).
- TC-INT-004 (approval bridge).
