# TC-INT-014 — Hosted MCP safe built-in call returns identical result via dispatch and CLI/HTTP/UDS

- **Priority:** P1
- **Type:** Integration / hosted MCP
- **Trace:** Task 10, Task 11

## Test Steps

1. Bind hosted MCP for a session permitted to call `agh__skill_view`.
2. Issue `tools/call agh__skill_view` over hosted MCP.
   - **Expected:** Returns the same content envelope as CLI/HTTP/UDS for the same input (modulo MCP-protocol packaging).
3. Verify telemetry includes one `tool.call_started`/`tool.call_completed` pair with `correlation_id` shared across hosted MCP and dispatch.

## Automation

- **Target:** Integration
- **Status:** Existing
- **Command/Spec:** `go test ./internal/mcp -run TestHostedSafeBuiltin`
