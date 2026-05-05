# TC-INT-013 — Hosted MCP `tools/list` equals `GET /api/sessions/{id}/tools`

- **Priority:** P0
- **Type:** Integration / hosted MCP parity
- **Trace:** Task 10, Task 11, Safety Invariant 13

## Objective

Prove hosted MCP `tools/list` is a strict projection of `GET /api/sessions/{id}/tools` for the same session.

## Test Steps

1. Establish ACP session with hosted MCP enabled.
2. Bind hosted MCP proxy.
3. Execute `tools/list` over hosted MCP and `GET /api/sessions/{id}/tools` simultaneously.
   - **Expected:** Same `tool_id` set in both responses.
4. Mutate state (disable extension); both refresh; both still match.
5. Divergence in any case is a Severity = High bug per Safety Invariant 13.

## Automation

- **Target:** Integration
- **Status:** Existing
- **Command/Spec:** `go test ./internal/mcp ./internal/api/core -run TestHostedMCPParity`
