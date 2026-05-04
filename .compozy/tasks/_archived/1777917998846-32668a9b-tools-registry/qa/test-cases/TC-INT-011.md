# TC-INT-011 — Local stdio MCP fixture lists and calls tool through `MCPCallExecutor`

- **Priority:** P0
- **Type:** Integration / MCP
- **Trace:** Task 09, ADR-010, ADR-011

## Test Steps

1. Configure `[mcp_servers.smoke_stdio]` with stdio transport pointing to a local MCP fixture (read-only `echo` and mutating `write_thing` tools).
2. Daemon discovers and registers tools as `mcp__smoke_stdio__echo` and `mcp__smoke_stdio__write_thing`.
3. Invoke `mcp__smoke_stdio__echo` via CLI/HTTP/UDS.
   - **Expected:** Successful call through `client.NewStdioMCPClient`.
4. Invoke `mcp__smoke_stdio__write_thing` without grant → denied.
5. Invoke with grant + approval → succeeds.
6. Telemetry shows `source.kind = mcp`, `source.owner = smoke_stdio`.

## Automation

- **Target:** Integration
- **Status:** Existing
- **Command/Spec:** `go test ./internal/mcp -run TestStdioMCPIntegration`
