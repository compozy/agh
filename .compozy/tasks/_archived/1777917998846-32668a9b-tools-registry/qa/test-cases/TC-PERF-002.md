# TC-PERF-002 — Hosted MCP projection-stream contention does not duplicate or drop deltas

- **Priority:** P2
- **Type:** Performance / concurrency
- **Trace:** Task 10, Safety Invariant 24

## Test Steps

1. Bind hosted MCP for one session.
2. In rapid succession (10 events / 1s): disable extension, re-enable, force MCP `expired`, MCP `authenticated`, change agent allow list.
3. Capture stream emissions and resulting `tools/list_changed` notifications.
   - **Expected:** Each terminal `tools/list` matches `GET /api/sessions/{id}/tools` for that moment; no duplicate adds; no missed removes.
4. Run with `-race`.

## Automation

- **Target:** Integration
- **Status:** Missing
- **Command/Spec:** `go test ./internal/mcp -run TestProjectionStreamContention -race`
