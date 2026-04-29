# TC-FUNC-002 — Tool pattern grammar: only exact IDs and namespace-prefix wildcards

- **Priority:** P1
- **Type:** Functional / pattern matching
- **Trace:** Task 02, Task 03, ADR-007

## Objective

Prove the policy/agent pattern grammar accepts only `<ToolID>` exact matches and `<prefix>__*` namespace wildcards. All other forms (regex, suffix wildcards, mid-segment wildcards, `*__view`, etc.) are rejected at config load time.

## Test Steps

1. Configure agent with `tools = ["agh__skill_view", "agh__skill_*", "mcp__github__*"]`.
   - **Expected:** Loads cleanly.
2. Configure with `tools = ["*__view"]`.
   - **Expected:** Reject; deterministic config error.
3. Configure with `tools = ["agh__*__view"]`.
   - **Expected:** Reject (mid-segment wildcard).
4. Configure with `tools = ["regex:.*"]`.
   - **Expected:** Reject (regex).
5. Configure with `toolsets = ["agh__skill_*"]`.
   - **Expected:** Reject — wildcards not allowed in `toolsets`; only `ToolsetID`.
6. Confirm pattern matches on canonical `ToolID` only — display titles never match.

## Automation

- **Target:** Unit
- **Status:** Existing
- **Command/Spec:** `go test ./internal/config -run TestAgentToolPatternGrammar`
