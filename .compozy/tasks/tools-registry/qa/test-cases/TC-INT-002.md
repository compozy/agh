# TC-INT-002 — Operator projection vs session projection cross-surface parity

- **Priority:** P1
- **Type:** Integration / projections
- **Trace:** Task 03, Task 11, Task 12

## Test Steps

1. CLI `agh tool list -o json` (operator scope) ↔ HTTP `GET /api/tools` ↔ UDS list operation.
   - **Expected:** Same payload (modulo `correlation_id`/timestamps).
2. CLI session-scope ↔ HTTP `GET /api/sessions/{id}/tools` ↔ Hosted MCP `tools/list` (when bound to that session).
   - **Expected:** Same callable subset.
3. Mutate state (disable extension); re-query.
   - **Expected:** All three surfaces converge after refresh.

## Automation

- **Target:** Integration
- **Status:** Existing
- **Command/Spec:** `go test ./internal/api/core ./internal/cli -run TestProjectionParity`
