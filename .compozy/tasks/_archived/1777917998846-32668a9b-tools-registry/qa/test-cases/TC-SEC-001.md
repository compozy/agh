# TC-SEC-001 — `deny-all` blocks every executable backend at dispatch time

- **Priority:** P0
- **Type:** Security / policy
- **Trace:** Task 03 (policy), Task 04 (dispatch), ADR-005, Safety Invariants 1, 2, 4

## Objective

Prove that ACP `permissions.mode = "deny-all"` blocks invocation across `native_go`, `extension_host`, and `mcp` backends through `Registry.Call`, while operator surfaces still show diagnostics.

## Preconditions

- Fresh `AGH_HOME`; daemon started.
- Workspace/session has `permissions.mode = "deny-all"`.
- One read-only `native_go` tool (`agh__skill_view`), one TypeScript extension-host read-only tool, and one local stdio MCP read-only tool registered.

## Test Steps

1. `agh tool list -o json`.
   - **Expected:** All three tools appear with `availability` carrying `policy_denied` reason and `visible_to_session = false`, `visible_to_operator = true`.
2. `agh tool invoke agh__skill_view --input '{"id":"agh__bootstrap"}' -o json`.
   - **Expected:** Exit non-zero; structured error `{"code":"tool_denied","reason_codes":["policy_denied"],"denying_layer":"system_permission_mode"}`. No content payload.
3. Repeat invoke against the TS extension-host tool ID.
   - **Expected:** Same structure; denying layer is `system_permission_mode`.
4. Repeat invoke against the MCP tool ID.
   - **Expected:** Same structure; no upstream MCP request emitted (verify via `qa/logs/security/mcp-fixture.log`).
5. `GET /api/sessions/{id}/tools`.
   - **Expected:** Empty `tools` array because session-callable projection is empty under `deny-all`.

## Edge Cases

- Operator HTTP endpoint `GET /api/tools` returns the same denied diagnostics; confirm reason codes match CLI.
- Hooks must NOT be invoked because policy denial happens before pre-call hooks (per dispatch ordering).

## Automation

- **Target:** Integration
- **Status:** Existing for native_go denial; Missing for cross-backend uniform negative path
- **Command/Spec:** `go test ./internal/tools -run TestPolicyDenyAll`; extend to cover all three backends
- **Notes:** Adds confidence that no backend can short-circuit the policy gate.
