# TC-FUNC-006 — `SourceRef` preserves raw provenance without becoming alternate identity

- **Priority:** P1
- **Type:** Functional / provenance
- **Trace:** Task 01, Task 09, ADR-007

## Objective

Prove that `SourceRef` preserves raw external MCP/extension names (`raw_server_name`, `raw_tool_name`) for diagnostics and collision handling, while canonical `ToolID` remains the only identity used by registry, policy, hooks, telemetry, and dispatch.

## Test Steps

1. Register an MCP tool whose raw name is `Create-Issue` from server `GitHub`.
   - **Expected:** `id = mcp__github__create_issue`, `source.raw_server_name = "GitHub"`, `source.raw_tool_name = "Create-Issue"`.
2. Match a policy pattern `mcp__github__*` against the raw name.
   - **Expected:** Pattern matches the canonical ID; never the raw name.
3. Display title may be `Create Issue`; canonical ID remains stable.
4. Confirm two raw `(server, tool)` pairs that normalize to the same canonical ID surface `conflicted_sanitized_name` (per TC-FUNC-003).

## Automation

- **Target:** Unit
- **Status:** Existing
- **Command/Spec:** `go test ./internal/tools -run TestSourceRefPreservation`
