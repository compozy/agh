# TC-INT-001: Operator Vs Session Projection Divergence

**Priority:** P1 (High)
**Type:** Integration
**Status:** Not Run
**Estimated Time:** 25 minutes
**Created:** 2026-04-30
**Last Updated:** 2026-04-30

## Objective

Prove operator projection (`GET /api/tools`) and session projection (`GET /api/sessions/{id}/tools`) diverge correctly under deny, unavailable, hook-blocked, and source-health-blocked conditions. Operator views must include reason codes for every excluded tool. Session views must list only callable tools.

## Traceability

- Task: task_01.
- TechSpec: "Implementation Design", "Test Strategy → Unit Tests".
- ADR: ADR-002.
- Surfaces: `internal/tools/policy.go`, `internal/api/core/tools.go`, `internal/daemon/native_tools.go`.

## Preconditions

- Isolated `AGH_HOME`.
- One agent definition that explicitly denies `agh__bridges_status`.
- One MCP server forced to FAILED via mock; this disables the corresponding tool by source health.
- One hook that denies `agh__memory_search`.

## Test Steps

1. Capture operator projection:
   ```bash
   curl -s --unix-socket "$AGH_HOME/run/sock/uds.sock" "http://localhost/api/tools" | tee qa/logs/TC-INT-001/operator.json
   ```
2. Capture session projection for the configured agent:
   ```bash
   curl -s --unix-socket "$AGH_HOME/run/sock/uds.sock" "http://localhost/api/sessions/$SID/tools" | tee qa/logs/TC-INT-001/session.json
   ```
3. Compare sets:
   ```bash
   jq -r '.[] | "\(.id)\t\(.callable)\t\((.reason_codes // []) | join(","))"' \
     qa/logs/TC-INT-001/operator.json | sort > qa/logs/TC-INT-001/operator.tsv
   jq -r '.[] | "\(.id)\t\(.callable)\t\((.reason_codes // []) | join(","))"' \
     qa/logs/TC-INT-001/session.json | sort > qa/logs/TC-INT-001/session.tsv
   diff qa/logs/TC-INT-001/operator.tsv qa/logs/TC-INT-001/session.tsv | tee qa/logs/TC-INT-001/diff.txt
   ```
4. Assert:
   - Session projection contains only entries with `callable=true`.
   - Operator projection contains the same set of IDs PLUS the denied/unavailable ones with `reason_codes` populated.
   - For `agh__bridges_status`: operator has `deny_tools` reason; session omits.
   - For an MCP-backed tool with FAILED source: operator has `source_health` reason; session omits.
   - For `agh__memory_search`: operator has `hook_deny` reason; session omits.
5. Trigger one direct call from the session for a denied tool:
   ```bash
   curl -s -X POST --unix-socket "$AGH_HOME/run/sock/uds.sock" \
     "http://localhost/api/tools/agh__bridges_status/invoke" -d '{}' | tee qa/logs/TC-INT-001/dispatch-deny.json
   ```
   - **Expected:** Dispatch revalidates and rejects with the same reason code as the operator projection (`deny_tools` or equivalent). No call reaches the underlying handler.

6. Run focused Go tests:
   ```bash
   go test ./internal/tools -run "TestProjection|TestPolicy" -count=1 | tee qa/logs/TC-INT-001/tools-tests.log
   go test ./internal/api/core -run "TestTools" -count=1 | tee qa/logs/TC-INT-001/api-core-tests.log
   ```

## Evidence To Capture

- `operator.json`, `session.json`, `operator.tsv`, `session.tsv`, `diff.txt`, `dispatch-deny.json`.
- Test logs.

## Edge Cases And Variations

| Variation | Input | Expected Result |
|-----------|-------|-----------------|
| Hook narrowing tool to read-only mode | hook reload | Tool callable=false in session projection with `hook_narrowed`/equivalent reason |
| Multiple denial sources stacked | deny_tools + hook_deny | Operator view shows both reason codes |
| Tool only available to operator scope | operator-only descriptor | Operator projection lists with `availability=operator_only`; session projection omits |
| Approval-required tool | tool with `approval_required=true` and approval channel reachable | Session projection still lists (callable subject to approval); approval surfaces during `Registry.Call` |

## Channels Exercised

- HTTP/UDS `/api/tools`, `/api/sessions/{id}/tools`, `/api/tools/{id}/invoke`.
- Daemon policy resolver.

## Related Test Cases

- TC-FUNC-001 (default discovery overlay + per-call recompute).
- TC-INT-005 (hook denial / source-health denial reason codes).
- TC-INT-006 (cache invalidation matrix).
