# TC-FUNC-008: MCP Auth Status Tool Diagnostics

**Priority:** P0 (Critical)
**Type:** Functional
**Status:** Not Run
**Estimated Time:** 25 minutes
**Created:** 2026-04-30
**Last Updated:** 2026-04-30

## Objective

Prove `agh__mcp_auth_status` returns redacted diagnostics from `internal/mcp/auth/service.go` and never triggers OAuth flow side effects. Confirm `agh mcp auth login` and `agh mcp auth logout` remain operator-only and are referenced in diagnostics rather than exposed as tools.

## Traceability

- Task: task_10 (MCP Auth Status and Hosted MCP Projection Parity).
- TechSpec: "Hosted MCP", "Existing MCP Config And Auth", "Existing MCP Config And Auth Lifecycle".
- ADR: ADR-004.
- Surfaces: `internal/tools/builtin/mcp_auth.go`, `internal/tools/mcp.go`, `internal/mcp/auth/service.go`, `internal/api/core/conversions.go`, `internal/cli/mcp_auth.go`.

## Preconditions

- Isolated `AGH_HOME`.
- Two MCP servers configured:
  - `mcp-server-a`: connected and authenticated.
  - `mcp-server-b`: token expired (forced via mock OAuth server).
- Real provider home isolated from `~/.codex` via `PROVIDER_HOME` / `PROVIDER_CODEX_HOME`.

## Test Steps

1. **Healthy server status:**
   ```bash
   agh tool invoke agh__mcp_auth_status --input '{"server_name":"mcp-server-a"}' -o json \
     | tee qa/logs/TC-FUNC-008/tool-status-a.json
   agh mcp auth status --server mcp-server-a -o json | tee qa/logs/TC-FUNC-008/cli-status-a.json
   diff <(jq -S . qa/logs/TC-FUNC-008/tool-status-a.json) \
        <(jq -S . qa/logs/TC-FUNC-008/cli-status-a.json)
   ```
   - **Expected:** Both responses contain `status` (e.g., `connected`), `auth_type`, `client_id`, `scopes`, `expires_at`, `refreshable`, `token_present`. They MUST NOT contain access tokens, refresh tokens, PKCE verifiers, or client secrets.

2. **Expired server status:**
   ```bash
   agh tool invoke agh__mcp_auth_status --input '{"server_name":"mcp-server-b"}' -o json \
     | tee qa/logs/TC-FUNC-008/tool-status-b.json
   ```
   - **Expected:** Status is `expired` (or analogous). `Diagnostic` field cites how to repair via `agh mcp auth login` (operator surface), but does not run the flow.

3. **Verify login/logout are NOT tool-callable:**
   ```bash
   agh tool invoke agh__mcp_auth_login --input '{"server_name":"mcp-server-b"}' -o json \
     | tee qa/logs/TC-FUNC-008/tool-login-attempt.json
   agh tool invoke agh__mcp_auth_logout --input '{"server_name":"mcp-server-b"}' -o json \
     | tee qa/logs/TC-FUNC-008/tool-logout-attempt.json
   ```
   - **Expected:** Both calls fail with a deterministic `tool_not_found` (or equivalent) error because no such tool ID is registered. The catalog does not list `agh__mcp_auth_login` / `agh__mcp_auth_logout`.

4. **CLI login/logout still work:**
   ```bash
   agh mcp auth login --server mcp-server-b 2>&1 | tee qa/logs/TC-FUNC-008/cli-login.log
   agh mcp auth status --server mcp-server-b -o json | tee qa/logs/TC-FUNC-008/cli-status-b-after-login.json
   ```
   - **Expected:** Operator login flow proceeds. After login, `agh__mcp_auth_status` reports `connected`; tokens are not exposed. (Run only when the lab provides a controllable mock OAuth flow.)

5. **Settings UI parity:**
   ```bash
   curl -s --unix-socket "$AGH_HOME/run/sock/uds.sock" "http://localhost/api/settings/mcp" | tee qa/logs/TC-FUNC-008/settings-mcp.json
   ```
   - **Expected:** Each MCP server entry includes redacted `auth_status`. No tokens / codes / PKCE values. The `token_present` boolean appears as the only public diagnostic.

6. **Refresh side-effect prohibition:**
   ```bash
   agh tool invoke agh__mcp_auth_status --input '{"server_name":"mcp-server-a","refresh":true}' -o json \
     | tee qa/logs/TC-FUNC-008/tool-status-refresh.json
   ```
   - **Expected:** Either the input is rejected (no refresh path on the tool) or refresh is a no-op for the tool surface. Mock OAuth server logs MUST NOT show a refresh request triggered by this call.

7. Run focused Go tests:
   ```bash
   go test ./internal/tools/builtin -run "TestMCPAuth" -count=1 | tee qa/logs/TC-FUNC-008/builtin-tests.log
   go test ./internal/mcp/auth -count=1 | tee qa/logs/TC-FUNC-008/auth-tests.log
   go test ./internal/tools -run "TestMCP" -count=1 | tee qa/logs/TC-FUNC-008/tools-mcp-tests.log
   ```

## Evidence To Capture

- All `qa/logs/TC-FUNC-008/*.json` payloads.
- Mock OAuth server log under `qa/logs/TC-FUNC-008/mock-oauth.log` (must show no triggered refresh from Step 6).
- Test logs.

## Edge Cases And Variations

| Variation | Input | Expected Result |
|-----------|-------|-----------------|
| Server name does not exist | `{"server_name":"missing"}` | Deterministic `mcp_server_not_found` error |
| Server with auth disabled (no OAuth) | static auth | Status reports `not_required` / equivalent without leaking secret material |
| Auth type bearer token only | bearer | `token_present=true`, `auth_type="bearer"`, never the bearer token itself |
| Operator surface drift | running on a daemon where `agh mcp auth login` was deleted | Test SHOULD fail because operator path is required by ADR-004 |

## Channels Exercised

- Tool invoke (daemon native provider).
- CLI (`agh mcp auth status`).
- HTTP/UDS settings endpoint.
- Mock OAuth server.

## Related Test Cases

- TC-SEC-002 (MCP auth redaction sweep).
- TC-INT-003 (hosted MCP projection parity).
- TC-INT-004 (approval bridge).
