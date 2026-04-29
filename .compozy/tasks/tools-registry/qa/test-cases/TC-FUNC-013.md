# TC-FUNC-013 — Child session lineage subset enforcement

- **Priority:** P1
- **Type:** Functional / lineage
- **Trace:** Task 03, ADR-005, Safety Invariant 6

## Objective

Prove a child session can only receive a subset of parent concrete `ToolID` atoms after toolset expansion. Widening attempts fail; persisted lineage uses concrete IDs only.

## Test Steps

1. Parent session lineage: `["agh__skill_view", "agh__tool_list"]`.
2. Spawn child requesting `["agh__skill_view"]`.
   - **Expected:** Allowed; child session-callable subset = `{agh__skill_view}`.
3. Spawn child requesting `["agh__skill_view", "agh__network_send"]`.
   - **Expected:** Rejected — child cannot widen beyond parent.
4. Persist lineage; restart daemon; load child.
   - **Expected:** Lineage atoms remain concrete `ToolID`s; no wildcards re-introduced.
5. Lineage atoms in legacy dotted form fail validation per Delete Targets.

## Automation

- **Target:** Integration
- **Status:** Existing
- **Command/Spec:** `go test ./internal/store -run TestSessionLineageSubset`
