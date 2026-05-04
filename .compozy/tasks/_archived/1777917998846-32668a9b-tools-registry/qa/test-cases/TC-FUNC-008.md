# TC-FUNC-008 — Agent `tools` / `toolsets` / `deny_tools` validation

- **Priority:** P1
- **Type:** Functional / agent grammar
- **Trace:** Task 02, ADR-007

## Objective

Prove the agent definition accepts canonical IDs and namespace wildcards in `tools` and `deny_tools`, accepts only `ToolsetID` values in `toolsets`, and rejects the legacy `["*"]` default.

## Test Steps

1. Agent with `tools = ["agh__skill_view", "ext__linear__*"]`, `toolsets = ["agh__bootstrap"]`, `deny_tools = ["agh__network_send"]`.
   - **Expected:** Loads cleanly.
2. Agent with `tools = ["*"]`.
   - **Expected:** Rejected — legacy default removed.
3. Agent with `toolsets = ["agh__skill_*"]`.
   - **Expected:** Rejected (no wildcards in toolsets).
4. Agent with `deny_tools = ["mcp__github__*"]` overlapping with allowed `tools`.
   - **Expected:** Loads cleanly; deny narrows allows at session-projection time.

## Automation

- **Target:** Unit
- **Status:** Existing
- **Command/Spec:** `go test ./internal/config -run TestAgentToolsGrammar`
