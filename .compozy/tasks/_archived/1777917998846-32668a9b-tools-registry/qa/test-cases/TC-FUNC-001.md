# TC-FUNC-001 — Canonical `ToolID` validator accepts MVP examples and rejects forbidden forms

- **Priority:** P0
- **Type:** Functional / grammar
- **Trace:** Task 01, ADR-007

## Objective

Prove `ToolID` validation accepts every MVP example listed in TechSpec Data Models and ADR-007, and rejects each disallowed form with deterministic reason codes.

## Preconditions

- Daemon running (or `go test ./internal/tools` harness).

## Test Steps

1. Validate accept set: `agh__tool_list`, `agh__tool_search`, `agh__tool_info`, `agh__skill_list`, `agh__skill_search`, `agh__skill_view`, `agh__network_peers`, `agh__network_send`, `agh__task_list`, `agh__task_read`, `mcp__github__create_issue`, `ext__linear__search`.
   - **Expected:** All accepted.
2. Validate reject set: `agh.skill.view`, `Agh__Tool_List`, `agh-tool-list`, `agh__`, `__agh__tool`, `agh__tool__`, empty string, single segment without `__`, segment starting with digit, all-uppercase, mixed case, very long ID > 64 chars.
   - **Expected:** Each rejected with deterministic error: `id_too_long` for over-length, `invalid_segment` / `id_invalid` for the rest.
3. Validate Unicode/non-ASCII inputs.
   - **Expected:** Rejected (lowercase ASCII only).

## Edge Cases

- Wildcard pattern parsing (`agh__skill_*`) is a *pattern*, not a `ToolID`; the validator distinguishes the two contexts (covered by TC-FUNC-002).

## Automation

- **Target:** Unit
- **Status:** Existing
- **Command/Spec:** `go test ./internal/tools -run TestToolIDValidation`
