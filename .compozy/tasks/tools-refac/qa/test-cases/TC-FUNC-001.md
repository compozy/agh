# TC-FUNC-001: Default Discovery Overlay And Per-Call Policy Recompute

**Priority:** P0 (Critical)
**Type:** Functional
**Status:** Not Run
**Estimated Time:** 30 minutes
**Created:** 2026-04-30
**Last Updated:** 2026-04-30

## Objective

Prove that every agent receives `agh__bootstrap` and `agh__catalog` as default discovery toolsets unless effective policy denies them, and that effective policy is recomputed per `list/search/get/call` from current runtime state (agent definition, session lineage, source policy, availability state, hook outputs). Prove operator vs session projections diverge correctly and that dispatch revalidates regardless of projection caching.

## Traceability

- Task: task_01 (Dynamic Policy Input Resolver and Default Discovery Overlay).
- TechSpec: "Implementation Design", "Data-Model Field Rationale", "Test Strategy → Unit Tests".
- ADRs: ADR-001, ADR-002.
- Surfaces: `internal/tools/policy.go`, `internal/tools/builtin/toolsets.go`, `internal/daemon/native_tools.go`, `internal/api/core/tools.go`, `internal/daemon/hosted_mcp.go`.

## Preconditions

- Isolated `AGH_HOME` from `agh-qa-bootstrap` (unique daemon ports, unique `tmux-bridge` socket).
- A workspace with two agent definitions:
  - `agent-empty`: `tools=[]`, `toolsets=[]`, `deny_tools=[]`.
  - `agent-deny-bootstrap`: `tools=[]`, `toolsets=[]`, `deny_tools=["agh__tool_list"]`.
- A workspace with one source whose health is currently FAILED so policy can mark a known tool unavailable.

## Test Steps

1. Boot the daemon and capture the canonical built-in catalog:
   ```bash
   agh tool list -o json | tee qa/logs/TC-FUNC-001/tool-list-operator.json
   ```
   - **Expected:** Listing includes the canonical built-in surface from the TechSpec table (`agh__tool_list`, `agh__tool_search`, `agh__tool_info`, `agh__skill_*`, `agh__network_*`, `agh__task_*`, `agh__autonomy_*`, `agh__memory_*`, `agh__sessions_*`, `agh__workspace_*`, `agh__config_*`, `agh__hooks_*`, `agh__automation_*`, `agh__extensions_*`, `agh__mcp_auth_status`, `agh__observe_*`, `agh__bridges_*`).

2. Start a session bound to `agent-empty` and record the session ID:
   ```bash
   agh session start --agent agent-empty --workspace $WS_ID -o json | tee qa/logs/TC-FUNC-001/session-start.json
   ```

3. Capture the session-scoped tool projection:
   ```bash
   curl -s --unix-socket "$AGH_HOME/run/sock/uds.sock" \
     "http://localhost/api/sessions/$SID/tools" | tee qa/logs/TC-FUNC-001/session-tools.json
   ```
   - **Expected:** Set includes every member of the `agh__bootstrap` and `agh__catalog` toolsets (`agh__tool_list`, `agh__tool_search`, `agh__tool_info`, `agh__skill_list`, `agh__skill_search`, `agh__skill_view`).

4. Bind hosted MCP and snapshot `tools/list`:
   ```bash
   agh tool mcp --session $SID --bind-nonce $NONCE 2>&1 | tee qa/logs/TC-FUNC-001/hosted-mcp.log
   ```
   - **Expected:** The hosted MCP `tools/list` set equals the session projection from Step 3 (set + ordering + reason codes for non-callable tools must match).

5. Switch to `agent-deny-bootstrap` and re-issue Step 3:
   - **Expected:** `agh__tool_list` no longer appears in the session projection. `agh__tool_search` and `agh__tool_info` remain, because deny is path-specific. `agh__catalog` members remain.

6. Open the operator-scope tool projection:
   ```bash
   curl -s --unix-socket "$AGH_HOME/run/sock/uds.sock" "http://localhost/api/tools" | tee qa/logs/TC-FUNC-001/tools-operator.json
   ```
   - **Expected:** `agh__tool_list` is present with `availability=denied` and `reason_codes` mentioning the deny that hid it from the session view.

7. Mutate runtime state and re-issue Step 3 between each mutation, asserting projection deltas:
   - Reload an agent definition (toolsets change, no daemon restart).
   - Toggle a source health from FAILED to OK.
   - Reload a hook with a new deny rule for `agh__bridges_status`.
   - Change the MCP auth health (mark a server `EXPIRED`).
   - Mutate a config overlay path that affects tool policy.
   - **Expected:** Each mutation changes the session projection on the next call. Cached projections do not survive the corresponding invalidation key.

8. Force a `Registry.Call` for a denied tool:
   ```bash
   agh tool invoke agh__tool_list -o json
   ```
   - **Expected:** Call rejected with deterministic reason code from the policy evaluator (e.g., `deny_tools_match`) even if a stale projection had cached it as callable.

## Evidence To Capture

- `qa/logs/TC-FUNC-001/tool-list-operator.json`
- `qa/logs/TC-FUNC-001/session-start.json`
- `qa/logs/TC-FUNC-001/session-tools.json` (per agent variant)
- `qa/logs/TC-FUNC-001/tools-operator.json`
- `qa/logs/TC-FUNC-001/hosted-mcp.log`
- `qa/logs/TC-FUNC-001/projection-deltas.txt` (session projection diffs across mutations)

## Edge Cases And Variations

| Variation | Input | Expected Result |
|-----------|-------|-----------------|
| Empty agent definition + empty session lineage | `agent-empty`, no parent session | Default overlay applies; both bootstrap and catalog visible |
| Explicit `deny_tools` for full overlay | `agent-deny-bootstrap` extended to deny `agh__tool_list`, `agh__tool_search`, `agh__tool_info` | Bootstrap toolset disappears from session projection; operator projection still lists with denial reasons |
| Source-health DOWN for an MCP-backed tool | source health flipped via test hook | Tool becomes unavailable on session projection but visible to operator with `source_health` reason |
| Hook deny mid-session | hook reload introduces deny on a callable tool | Next session projection reflects the deny |
| Stale cache check | Projection cache hit, runtime mutated | Dispatch revalidates and rejects with deterministic reason |

## Channels Exercised

- HTTP/UDS (`/api/tools`, `/api/sessions/{id}/tools`, `/api/tools/{id}/invoke`).
- Hosted MCP `tools/list`.
- CLI (`agh tool list`, `agh tool invoke`).
- Daemon prompt assembly (verify default discovery survives prompt rebuild — record the rendered tools section under `qa/logs/TC-FUNC-001/prompt.txt`).

## Related Test Cases

- TC-INT-001 (operator vs session projection divergence).
- TC-INT-002 (transport parity).
- TC-INT-003 (hosted MCP equality).
- TC-INT-006 (cache invalidation matrix).
- TC-FUNC-002 (prompt section + bundled guide rendering).
