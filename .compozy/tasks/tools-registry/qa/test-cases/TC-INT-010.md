# TC-INT-010 — Go SDK mutating tool gated by policy/approval

- **Priority:** P2
- **Type:** Integration / Go SDK
- **Trace:** Task 08, ADR-005

## Test Steps

1. Go subprocess extension publishes mutating tool with explicit `destructive = true`.
2. Invoke without grant → denied.
3. With explicit grant + approval token → succeeds.
4. Test `go-tool-provider` create-extension scaffold builds and runs.

## Automation

- **Target:** Integration
- **Status:** Existing
- **Command/Spec:** `go test ./sdk/go -run TestGoMutatingTool`
