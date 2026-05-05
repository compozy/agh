# TC-INT-008 — TypeScript mutating extension tool gated by policy/approval

- **Priority:** P2
- **Type:** Integration / extension-host
- **Trace:** Task 07, ADR-005

## Test Steps

1. Fixture extension publishes mutating tool (`destructive = true`).
2. `permissions.mode = "approve-reads"`; tool requires explicit grant + approval.
3. Invoke without grant → `policy_denied`.
4. Add explicit `tools = ["ext__ts_test__write_thing"]`, mint approval token, replay.
   - **Expected:** Successful call.
5. Hosted MCP variant: approval bridge fires (TC-FUNC-048).

## Automation

- **Target:** Integration
- **Status:** Existing
- **Command/Spec:** `go test ./internal/extension -run TestTSMutatingTool`
