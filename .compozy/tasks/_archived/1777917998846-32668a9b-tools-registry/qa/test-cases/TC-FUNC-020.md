# TC-FUNC-020 — `agh__tool_info` returns descriptor + availability + source provenance

- **Priority:** P2
- **Type:** Functional / native tool
- **Trace:** Task 05, ADR-004

## Test Steps

1. Invoke with valid `tool_id`.
   - **Expected:** Returns descriptor, availability, source provenance, and policy-view fields. No tokens.
2. Invoke with invalid `tool_id`.
   - **Expected:** Schema validation error or `tool_not_found`.
3. Invoke with hidden tool from a session context.
   - **Expected:** `tool_not_found` from session view; operator view shows the descriptor + reasons.

## Automation

- **Target:** Unit
- **Status:** Existing
- **Command/Spec:** `go test ./internal/tools -run TestNativeAghToolInfo`
