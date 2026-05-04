# TC-FUNC-028 — Schema validation rejects malformed input before provider invocation

- **Priority:** P1
- **Type:** Functional / dispatch
- **Trace:** Task 04, Safety Invariant 1

## Test Steps

1. Provider configured with strict object schema.
2. Invoke with primitive (`42`).
   - **Expected:** `tool_invalid_input` reason `schema_invalid`; provider not called.
3. Invoke with extra unknown property when schema disallows.
   - **Expected:** Same.
4. Invoke with property of wrong type.
   - **Expected:** Same; error message lists offending path.

## Automation

- **Target:** Unit
- **Status:** Existing
- **Command/Spec:** `go test ./internal/tools -run TestDispatchSchemaValidation`
