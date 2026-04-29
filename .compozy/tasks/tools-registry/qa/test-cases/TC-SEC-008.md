# TC-SEC-008 — `cloneDaemonMCPServer` preserves `Transport`/`URL`/`Auth`

- **Priority:** P1
- **Type:** Security / config preservation
- **Trace:** Task 09, ADR-010, TechSpec implementation correction

## Objective

Prove that the resource-cloning path used by `internal/daemon/tool_mcp_resources.go` preserves remote MCP `Transport`, `URL`, and `Auth` (including `MCPAuthConfig` metadata) end-to-end. A clone that strips remote fields would silently drop OAuth metadata from tool diagnostics.

## Preconditions

- `[mcp_servers.fake_http]` configured with `transport = "http"`, `url`, and `MCPAuthConfig` (issuer, authorization, token, scopes, `client_id`, `client_secret_env`).
- `[mcp_servers.fake_sse]` configured with `transport = "sse"`.

## Test Steps

1. Daemon load.
2. Inspect projected MCP server resource via `GET /api/settings/mcp-servers`.
   - **Expected:** Both servers retain `transport`, `url`, and (for HTTP) `auth.metadata_url`/`scopes`/`client_id`. SSE retains `transport = "sse"` (not silently rewritten to `http`).
3. Inspect tool diagnostics via `agh tool info mcp__fake_http__<t>`.
   - **Expected:** Auth status mirrors settings; no token material; reasons reflect current state.
4. Re-load daemon with a config that omits `transport`.
   - **Expected:** Validation rejects the config with deterministic error (no silent default).

## Edge Cases

- Cloning a server with `transport = "stdio"` and command/args/env retains all four fields without leaking env values into tool diagnostics.
- Skill-sidecar MCP entries (stdio-only) preserve `command`/`args`/`env` and DO NOT gain phantom auth fields.

## Automation

- **Target:** Unit + Integration
- **Status:** Existing
- **Command/Spec:** `go test ./internal/daemon -run TestCloneMCPServerPreservesTransportAuth`
