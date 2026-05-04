# TC-INT-004: Hosted MCP Approval Bridge Under Timeout, Cancel, And Disconnect

**Priority:** P0 (Critical)
**Type:** Integration
**Status:** Not Run
**Estimated Time:** 30 minutes
**Created:** 2026-04-30
**Last Updated:** 2026-04-30

## Objective

Prove the hosted MCP approval bridge:

- Routes approval-required tool calls through ACP `session/request_permission`.
- Surfaces the result inside `tools/call` (approve/deny/timeout/cancel/disconnect).
- Emits deterministic reason codes (`approval_required`, `approval_unreachable`, `approval_timed_out`, `approval_canceled`).
- Cannot be satisfied by client-supplied approval credentials (no `approval_token`, no inline approval payload).

## Traceability

- Task: task_10.
- TechSpec: "Hosted MCP → approval bridge", "Safety Invariants".
- ADR: ADR-002.
- Surfaces: `internal/daemon/tool_approval_bridge.go`, `internal/mcp/hosted.go`, `internal/tools/policy.go`.

## Preconditions

- Isolated `AGH_HOME`.
- A controllable mock ACP approval channel that can: (a) approve, (b) deny, (c) hang, (d) be programmatically disconnected.
- A session with at least one tool whose descriptor has `approval_required=true` (any mutating built-in such as `agh__config_set` qualifies).
- Hosted MCP bind achievable.

## Test Steps

1. **Approve path:**
   - From hosted MCP, issue `tools/call` for the approval-required tool.
   - Approve via mock ACP.
   - **Expected:** Call succeeds; response carries the tool result. The approval bridge log shows one `session/request_permission` round-trip.

2. **Deny path:**
   - Issue same call; deny via mock ACP.
   - **Expected:** Call fails with `error.code=approval_required` and reason `approval_denied`. Tool was not executed (inspect daemon logs / handler counters).

3. **Timeout path:**
   - Configure `[tools.policy].approval_timeout_seconds=2`.
   - Issue same call; mock ACP does not respond.
   - **Expected:** Within ~2 seconds, call fails with `approval_required` plus `approval_timed_out`.

4. **Hosted MCP disconnect:**
   - Issue same call; before approval comes back, drop the hosted MCP transport.
   - **Expected:** Approval context is canceled. When the next call is attempted (or via daemon log), reason includes `approval_canceled`.

5. **ACP unavailable:**
   - Make the ACP approval channel unreachable.
   - Issue the call.
   - **Expected:** Hosted MCP `tools/list` (refreshed) hides the tool. If the call still reaches dispatch, response is `approval_required` plus `approval_unreachable`.

6. **Client-supplied credentials rejected:**
   - Issue the call with `arguments` containing `approval_token`, `approval`, or any synthetic approval credential.
   - **Expected:** Bridge ignores the field; ACP `session/request_permission` is still issued; the approval credential cannot satisfy the bridge.

7. **CLI / HTTP / UDS comparison:**
   - For the same call surfaced via CLI or HTTP/UDS with `approval_token=<known-good>`, confirm those surfaces accept `approval_token` (per TechSpec wording: "CLI/HTTP/UDS may use `approval_token`, hosted MCP must use the daemon approval bridge").
   - **Expected:** Hosted MCP behavior (Step 6) and the non-MCP surface behavior diverge as documented; this proves the bridge enforces the boundary.

8. Run focused Go tests:
   ```bash
   go test ./internal/daemon -run "TestApproval|TestHostedMCP" -count=1 | tee qa/logs/TC-INT-004/daemon-tests.log
   go test ./internal/tools -run "TestApproval" -count=1 | tee qa/logs/TC-INT-004/tools-tests.log
   ```

## Evidence To Capture

- Mock ACP server log under `qa/logs/TC-INT-004/mock-acp.log`.
- Hosted MCP frames under `qa/logs/TC-INT-004/mcp-frames.jsonl` for each path.
- Daemon log filtered for `approval` events under `qa/logs/TC-INT-004/daemon-approval.log`.
- Test logs.

## Edge Cases And Variations

| Variation | Input | Expected Result |
|-----------|-------|-----------------|
| Approval channel toggles mid-call | flap reachability | Outcome is deterministic per the snapshot used at request time |
| Two concurrent approval-required calls in same session | parallel `tools/call` | Each call gets its own `session/request_permission`; no shared approval re-use |
| Tool whose policy denies first | deny + approval gating | Policy denial wins (deterministic reason); approval bridge not reached |
| Permission mode = `auto` with reachable channel | auto-approve | Bridge approves immediately; reason includes `auto_approved` if the bridge supports it |

## Channels Exercised

- Hosted MCP JSON-RPC.
- ACP `session/request_permission`.
- Approval bridge log.

## Related Test Cases

- TC-INT-003 (hosted MCP `tools/list` parity).
- TC-FUNC-008 (MCP auth status).
- TC-SEC-006 (hosted MCP bind safety).
