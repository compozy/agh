# TC-FUNC-056 — `agh toolsets list/info` renders expanded members

- **Priority:** P2
- **Type:** Functional / CLI
- **Trace:** Task 12

## Test Steps

1. `agh toolsets list -o json`.
   - **Expected:** All known toolsets with canonical `ToolsetID`, member count, and any conflicts.
2. `agh toolsets info agh__bootstrap -o json`.
   - **Expected:** Concrete `ToolID` expansion, including unavailable members with reason codes.
3. Invoking a toolset is not supported (toolsets are for policy/lineage).
4. Unknown toolset returns deterministic `tool_not_found` analog.

## Automation

- **Target:** Integration
- **Status:** Existing
- **Command/Spec:** `go test ./internal/cli -run TestToolsetsCommand`
