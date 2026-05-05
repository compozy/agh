# TC-FUNC-038 — Local stdio MCP supports `tools/list` and `tools/call`

- **Priority:** P1
- **Type:** Functional / MCP
- **Trace:** Task 09, ADR-010

## Test Steps

1. Configure a local stdio MCP fixture with one read-only and one mutating tool.
2. Daemon discovers and lists both tools.
3. Invoke read-only tool through `Registry.Call`.
   - **Expected:** Successful call.
4. Invoke mutating tool with no policy grant.
   - **Expected:** `policy_denied`.
5. Invoke mutating tool with explicit grant + approval.
   - **Expected:** Successful call.
6. Confirm `mcp__<server>__<tool>` canonical IDs.

## Automation

- **Target:** Integration
- **Status:** Existing
- **Command/Spec:** `go test ./internal/mcp -run TestStdioMCPCallThrough`
