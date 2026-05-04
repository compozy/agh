# TC-FUNC-037 — Remote HTTP MCP uses streamable HTTP client; SSE uses explicit SSE client

- **Priority:** P1
- **Type:** Functional / transport
- **Trace:** Task 09, ADR-010, ADR-011, Safety Invariant 26

## Objective

Prove remote HTTP transport calls go through `client.NewStreamableHttpClient` and remote SSE transport calls go through `client.NewSSEMCPClient`. Neither is silently rewritten into the other.

## Test Steps

1. `[mcp_servers.fake_http].transport = "http"`. Invoke a tool.
   - **Expected:** Streamable HTTP client used. Network capture shows streaming HTTP request shape (not SSE).
2. `[mcp_servers.fake_sse].transport = "sse"`. Invoke a tool.
   - **Expected:** Library SSE client used. Capture shows SSE-specific behavior.
3. Attempt to set `transport = "websocket"`.
   - **Expected:** Config validation rejects.
4. Code grep: no occurrence of `transport = "http"` re-mapping `sse` or vice versa in `internal/mcp`.

## Automation

- **Target:** Integration
- **Status:** Existing
- **Command/Spec:** `go test ./internal/mcp -run TestRemoteTransportSelection`
