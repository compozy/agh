# TC-SEC-012 — Hosted MCP rejects client-supplied approval tokens

- **Priority:** P0
- **Type:** Security / approval boundary
- **Trace:** Task 10, ADR-002, ADR-005, Safety Invariants 17, 21

## Objective

Prove that hosted MCP `tools/call` cannot satisfy `approval_required` using client-supplied arguments. Approval must come from the daemon-mediated approval bridge using ACP `session/request_permission`.

## Preconditions

- Hosted MCP proxy bound to a session with `permissions.mode = "approve-reads"` and a mutating tool requiring approval.

## Test Steps

1. Issue `tools/call` for the mutating tool with no input fields suggesting approval.
   - **Expected:** Daemon issues ACP permission request; awaits user approval; on approval, executes the tool.
2. Re-issue `tools/call` with `approval_token` field present in the input.
   - **Expected:** Field is treated as ordinary input (subject to schema validation) and is NOT consumed as approval; approval still flows through the bridge.
3. Issue `tools/call` with a stolen valid approval token in input.
   - **Expected:** Same — the token is not honored; tool either schema-validates the field or rejects.
4. ACP permission denied.
   - **Expected:** Hosted MCP returns `tool_denied` reason `policy_denied` mapped from ACP denial.
5. ACP permission times out beyond `[tools.policy].approval_timeout_seconds`.
   - **Expected:** Hosted MCP returns `tool_approval_required` with reasons `approval_required` and `approval_timed_out`.

## Edge Cases

- Approval channel unavailable (no ACP `session/request_permission` available).
   - **Expected:** Tool hidden from `tools/list` if knowable at projection time; if call still arrives, response is `approval_required` + `approval_unreachable`.
- Hosted MCP request deadline must be at least `approval_timeout_seconds + 5s` so transport timeout cannot preempt valid approval wait.

## Automation

- **Target:** Integration
- **Status:** Existing
- **Command/Spec:** `go test ./internal/mcp -run TestHostedMCPApprovalBridge`
