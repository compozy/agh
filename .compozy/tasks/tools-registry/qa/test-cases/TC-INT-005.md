# TC-INT-005 — End-to-end native tool invocation from CLI through registry to skills service

- **Priority:** P1
- **Type:** Integration / native dispatch
- **Trace:** Task 05, Task 11, Task 12

## Test Steps

1. CLI: `agh tool invoke agh__skill_view --input '{"id":"agh__bootstrap"}' -o json`.
   - **Expected:** Daemon UDS handler calls `Registry.Call`; native provider calls `internal/skills.Registry`; result returned with truncation metadata if oversized.
2. HTTP equivalent issues same call.
3. Telemetry: `tool.call_completed` recorded with `source.kind = builtin`.
4. Verify hooks fire in correct order.

## Automation

- **Target:** Integration
- **Status:** Existing
- **Command/Spec:** `go test ./internal/cli ./internal/tools -run TestE2ENativeSkillView`
