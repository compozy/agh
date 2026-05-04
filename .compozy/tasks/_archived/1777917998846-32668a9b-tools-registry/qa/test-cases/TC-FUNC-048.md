# TC-FUNC-048 — Hosted MCP approval bridge times out with `approval_timed_out`

- **Priority:** P1
- **Type:** Functional / approval bridge
- **Trace:** Task 10, ADR-005, Safety Invariants 17, 25

## Test Steps

1. `[tools.policy].approval_timeout_seconds = 10`.
2. Mutating tool requires approval; ACP permission request stalls.
3. Hosted MCP `tools/call` issued.
   - **Expected:** Daemon waits up to 10s; library-facing request deadline ≥ 15s (5s guard band); on timeout the call returns `approval_required` + `approval_timed_out`.
4. Verify hosted MCP request deadline did NOT preempt the wait — confirm via timing capture.
5. Remote MCP outbound deadlines remain independent.

## Automation

- **Target:** Integration
- **Status:** Existing
- **Command/Spec:** `go test ./internal/mcp -run TestApprovalBridgeTimeout`
