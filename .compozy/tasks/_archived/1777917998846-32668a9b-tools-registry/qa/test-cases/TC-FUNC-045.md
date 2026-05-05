# TC-FUNC-045 — Hosted MCP exposes only AGH-hosted stdio entry; never remote MCP servers

- **Priority:** P0
- **Type:** Functional / hosted MCP injection
- **Trace:** Task 10, ADR-002, ADR-010, Safety Invariant 13

## Objective

Prove ACP `mcpServers` injection includes only the AGH-hosted stdio MCP entry. Configured remote HTTP/SSE/stdio MCP servers remain daemon-owned registry backends and are NOT injected directly into ACP `mcpServers`.

## Test Steps

1. Configure remote HTTP MCP server `fake_http` and SSE server `fake_sse`.
2. Start session via ACP runtime.
3. Capture ACP session payload `mcpServers`.
   - **Expected:** Single entry pointing to `agh tool mcp --session ... --bind-nonce ...` over stdio. No remote URLs.
4. Confirm `toSDKMCPServers` (or replacement) does NOT convert remote HTTP/SSE into blank stdio entries.

## Automation

- **Target:** Integration
- **Status:** Existing
- **Command/Spec:** `go test ./internal/acp -run TestACPInjectionHostedOnly`
