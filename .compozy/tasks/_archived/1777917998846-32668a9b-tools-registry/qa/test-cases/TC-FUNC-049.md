# TC-FUNC-049 — Hosted MCP proxy disconnect cancels in-flight call with `approval_canceled`

- **Priority:** P1
- **Type:** Functional / approval bridge
- **Trace:** Task 10, Safety Invariant 17

## Test Steps

1. Mutating tool requires approval.
2. Hosted MCP `tools/call` issued; user has not approved yet.
3. Disconnect the proxy stdio.
   - **Expected:** Derived context canceled; call returns `approval_required` + `approval_canceled`; no stale launch records; ACP permission request canceled.
4. Subsequent calls require fresh bind nonce (covered by TC-SEC-010).

## Automation

- **Target:** Integration
- **Status:** Existing
- **Command/Spec:** `go test ./internal/mcp -run TestHostedDisconnectCancel`
