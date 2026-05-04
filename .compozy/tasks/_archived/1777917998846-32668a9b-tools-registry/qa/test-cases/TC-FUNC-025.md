# TC-FUNC-025 — `agh__task_child_create` cannot widen child lineage beyond parent

- **Priority:** P1
- **Type:** Functional / lineage authority
- **Trace:** Task 05, ADR-004, Safety Invariant 18

## Objective

Prove `agh__task_child_create` calls `task.Service.CreateChildTask`. Lineage subset enforcement remains in service-level authority; registry policy may narrow but cannot widen.

## Test Steps

1. Parent task lineage: `["agh__skill_view"]`.
2. Invoke `agh__task_child_create` with child requesting `["agh__skill_view"]`.
   - **Expected:** Child created with subset.
3. Invoke with child requesting `["agh__skill_view", "agh__network_send"]`.
   - **Expected:** Service returns lineage error; tool returns `tool_denied` reason `session_denied`.
4. Confirm tool result envelope does not leak full parent lineage; only the child decision is exposed.

## Automation

- **Target:** Integration
- **Status:** Existing
- **Command/Spec:** `go test ./internal/task -run TestCreateChildTaskLineageSubset`
