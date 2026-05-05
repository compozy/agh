# TC-FUNC-055 — `agh tool invoke` validates JSON input and redacts sensitive fields

- **Priority:** P1
- **Type:** Functional / CLI
- **Trace:** Task 12, Safety Invariant 11

## Test Steps

1. `agh tool invoke agh__skill_view --input '{"id":"agh__bootstrap"}' -o json`.
   - **Expected:** Returns content with truncation metadata if applicable.
2. Pass invalid JSON via `--input`.
   - **Expected:** Exit non-zero; CLI emits structured `code=invalid_input`.
3. Pass JSON via `--input-file path` and stdin.
   - **Expected:** Both paths supported.
4. Sensitive field in input (e.g. `tools.sensitive.field:LEAK_v1`).
   - **Expected:** Result envelope redacts the field; stdout/stderr never echo the sentinel.

## Automation

- **Target:** Integration
- **Status:** Existing
- **Command/Spec:** `go test ./internal/cli -run TestToolInvokeCommand`
