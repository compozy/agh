# TC-INT-003: Hosted MCP `tools/list` Equals Session Projection

**Priority:** P0 (Critical)
**Type:** Integration
**Status:** Not Run
**Estimated Time:** 25 minutes
**Created:** 2026-04-30
**Last Updated:** 2026-04-30

## Objective

Prove the hosted MCP `tools/list` projection equals `GET /api/sessions/{id}/tools` exactly: same set, same ordering, same approval-required flags, same reason codes for absent tools. Confirm the projection updates after policy mutations between binds.

## Traceability

- Task: task_10 (and inherited from task_01).
- TechSpec: "Hosted MCP", "Safety Invariants".
- ADR: ADR-002.
- Surfaces: `internal/daemon/hosted_mcp.go`, `internal/mcp/hosted.go`, `internal/api/core/sessions.go`, `internal/tools/policy.go`.

## Preconditions

- Isolated `AGH_HOME`.
- Two sessions: `S_A` (default agent), `S_B` (agent denying `agh__memory_search` and `agh__bridges_status`).
- Hosted MCP bind achievable for both sessions.

## Test Steps

1. Bind hosted MCP for `S_A` and capture `tools/list` JSON-RPC response:
   ```bash
   # the bind harness records JSON-RPC frames
   agh tool mcp --session $S_A --bind-nonce $NONCE_A 2>&1 | tee qa/logs/TC-INT-003/bind-a.log
   ```
   Save the response under `qa/logs/TC-INT-003/mcp-tools-list-a.json` and extract the tool list.

2. Capture HTTP/UDS session projection for `S_A`:
   ```bash
   curl -s --unix-socket "$AGH_HOME/run/sock/uds.sock" "http://localhost/api/sessions/$S_A/tools" \
     | tee qa/logs/TC-INT-003/session-a.json
   ```

3. Compare:
   ```bash
   jq -r '.result.tools[] | .name' qa/logs/TC-INT-003/mcp-tools-list-a.json | nl > qa/logs/TC-INT-003/mcp-a.tsv
   jq -r '.[] | select(.callable==true) | .id' qa/logs/TC-INT-003/session-a.json | nl > qa/logs/TC-INT-003/api-a.tsv
   diff qa/logs/TC-INT-003/mcp-a.tsv qa/logs/TC-INT-003/api-a.tsv | tee qa/logs/TC-INT-003/diff-a.txt
   ```
   - **Expected:** Empty diff. Order matches exactly.

4. Repeat steps 1-3 for `S_B`. Confirm `agh__memory_search` and `agh__bridges_status` are absent from both surfaces.

5. **Mutation between binds:** mutate the agent definition for `S_A` to deny `agh__network_send`. Re-bind hosted MCP and re-capture `tools/list`:
   - **Expected:** New `tools/list` no longer contains `agh__network_send`. Session projection agrees.

6. **Approval-required tools:** if any tool has `approval_required=true`, confirm:
   - It still appears in session projection and hosted MCP `tools/list`.
   - The hosted MCP descriptor (annotation/`metadata`) marks it appropriately so models know an approval will be required.
   - If approval channel is unreachable for the session, the tool is hidden from hosted MCP `tools/list` but still visible to operator with `approval_unreachable` reason.

7. **No raw `claim_token` in `tools/list`:**
   ```bash
   grep -nE "claim_token" qa/logs/TC-INT-003/mcp-tools-list-*.json
   ```
   - **Expected:** Zero matches.

8. Run focused Go tests:
   ```bash
   go test ./internal/daemon -run "TestHostedMCP" -count=1 | tee qa/logs/TC-INT-003/daemon-tests.log
   go test ./internal/mcp -run "TestHosted" -count=1 | tee qa/logs/TC-INT-003/mcp-tests.log
   ```

## Evidence To Capture

- `mcp-tools-list-{a,b}.json`, `session-{a,b}.json`, diffs, mutation re-bind logs, approval annotation samples.
- Test logs.

## Edge Cases And Variations

| Variation | Input | Expected Result |
|-----------|-------|-----------------|
| Tool descriptor change between binds | extension reload | Subsequent `tools/list` reflects new descriptor; previous bind's frames are not authoritative |
| Approval channel disconnects mid-session | force ACP disconnect | Hosted MCP `tools/list` projection refreshes; tools with unreachable approval drop |
| Session's hosted MCP closes | proxy disconnect | Bind invalidated; further `tools/list` requires a new bind |
| Foreign session ID supplied after bind | the daemon must reject any `session_id` swap | `permission_denied` |

## Channels Exercised

- Hosted MCP JSON-RPC.
- HTTP/UDS session projection.
- Daemon hosted MCP exposure.

## Related Test Cases

- TC-FUNC-001 (default discovery overlay).
- TC-INT-002 (transport parity).
- TC-INT-004 (approval bridge).
- TC-SEC-006 (hosted MCP bind safety).
