# TC-INT-015 — Mid-session degradation propagates to hosted MCP via projection stream

- **Priority:** P2
- **Type:** Integration / hosted MCP / lifecycle
- **Trace:** Task 10, Safety Invariants 13, 24

## Test Steps

1. Bind hosted MCP for a session containing native + TS extension + MCP tools.
2. Disable the TS extension mid-session.
   - **Expected:** Stream emits remove; hosted MCP fires `tools/list_changed`; new `tools/list` matches refreshed `GET /api/sessions/{id}/tools`.
3. Force MCP server `expired`.
   - **Expected:** MCP-backed tool removed from session projection; hosted MCP refreshes accordingly.
4. Drop the projection stream entirely.
   - **Expected:** Proxy fail-closes by closing the MCP session.

## Automation

- **Target:** Integration
- **Status:** Existing
- **Command/Spec:** `go test ./internal/mcp -run TestMidSessionDegradation`
