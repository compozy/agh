# TC-FUNC-047 — Hosted MCP projection stream drives `tools/list_changed`

- **Priority:** P1
- **Type:** Functional / hosted MCP lifecycle
- **Trace:** Task 10, Safety Invariant 24

## Objective

Prove the daemon→proxy projection stream propagates mid-session changes (extension disable, MCP auth degradation, policy change) into `mcp-go` add/remove/replace operations and emits `notifications/tools/list_changed`. `tools/list` stays equal to `GET /api/sessions/{id}/tools`.

## Test Steps

1. Establish hosted MCP bound session with 5 callable tools.
2. Disable an extension that contributed 1 tool.
   - **Expected:** Stream emits remove; library fires `tools/list_changed`; `tools/list` returns 4 tools matching `GET /api/sessions/{id}/tools`.
3. Force MCP auth `expired`.
   - **Expected:** MCP-backed tool removed from session projection; same parity.
4. Drop the projection stream (simulate failure).
   - **Expected:** Proxy fail-closes by closing the MCP session rather than serving stale tool catalog.

## Automation

- **Target:** Integration
- **Status:** Existing
- **Command/Spec:** `go test ./internal/mcp -run TestHostedProjectionStream`
