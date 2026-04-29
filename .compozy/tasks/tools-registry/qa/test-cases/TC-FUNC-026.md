# TC-FUNC-026 — Native descriptors classify risk flags accurately

- **Priority:** P1
- **Type:** Functional / risk classification
- **Trace:** Task 05, ADR-004, ADR-005

## Objective

Prove every native descriptor classifies `read_only`, `destructive`, `open_world`, `requires_interaction`, `concurrency_safe`, and `risk` accurately per ADR-004 tables.

## Test Steps

For each native tool:

| ToolID | read_only | destructive | open_world |
|--------|-----------|-------------|------------|
| agh__tool_list/search/info | true | false | false |
| agh__skill_list/search/view | true | false | false |
| agh__network_peers | true | false | false |
| agh__network_send | false | false | true |
| agh__task_list/read/run_list | true | false | false |
| agh__task_create/child_create/update | false | false | false |
| agh__task_cancel | false | true | false |

1. Invoke `agh__tool_info` for each id.
   - **Expected:** Flags match the table.
2. A risk-flag mismatch is a Severity = High bug.

## Automation

- **Target:** Unit
- **Status:** Existing
- **Command/Spec:** `go test ./internal/tools -run TestNativeRiskClassification`
