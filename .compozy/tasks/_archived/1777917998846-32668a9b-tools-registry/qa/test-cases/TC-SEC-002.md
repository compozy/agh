# TC-SEC-002 — `approve-reads` does not auto-approve untrusted external read-only tools

- **Priority:** P0
- **Type:** Security / policy
- **Trace:** Task 02 (`trusted_sources`), Task 03, ADR-005, Safety Invariant 5

## Objective

Prove that `approve-reads` only auto-approves AGH-classified `read_only` `native_go` tools and any `extension`/`mcp` source listed under `[tools.policy].trusted_sources`. Untrusted read-only extension/MCP tools require explicit grant or approval.

## Preconditions

- `permissions.mode = "approve-reads"`.
- `[tools.policy].trusted_sources = []`.
- One TypeScript extension `read_only` tool (untrusted source) registered.
- One MCP `read_only` tool from an untrusted server registered.
- One AGH-native `read_only` tool (`agh__tool_list`) available.

## Test Steps

1. `agh tool invoke agh__tool_list --input '{}' -o json`.
   - **Expected:** Returns `200`, no approval prompt.
2. Invoke the untrusted TypeScript extension read-only tool.
   - **Expected:** `approval_required` reason code; CLI prompts or returns `tool_approval_required` when no `--approval-token` provided.
3. Add the extension to `trusted_sources = ["extension:test_ext"]`, restart daemon.
4. Re-invoke the extension tool.
   - **Expected:** Auto-approves, returns content.
5. Invoke the untrusted MCP tool.
   - **Expected:** Still `approval_required`; trusted_sources entry is per-source, not global.
6. Add `trusted_sources = ["extension:test_ext", "mcp:smoke_stdio"]`.
7. Re-invoke MCP tool.
   - **Expected:** Auto-approves.

## Edge Cases

- Mutating extension tool with `read_only = false` MUST NOT auto-approve regardless of `trusted_sources` (proven by TC-SEC-003).
- `trusted_sources` entries that do not resolve to a known extension/MCP source must fail config validation at load time (covered by TC-FUNC-014).

## Automation

- **Target:** Integration
- **Status:** Existing partially; Missing combined matrix
- **Command/Spec:** `go test ./internal/tools -run TestPolicyApproveReads`
- **Notes:** Critical because it directly expresses the trust boundary for external read-only tools.
