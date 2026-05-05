# TC-FUNC-018 — `agh__tool_list` lists registered tools with availability

- **Priority:** P1
- **Type:** Functional / native tool
- **Trace:** Task 05, ADR-004

## Objective

Prove `agh__tool_list` returns the operator/session projection it is invoked under, with canonical `tool_id`, `display_title`, `source.kind`, `risk`, `read_only`, and `availability` reason codes.

## Test Steps

1. From an operator-scoped CLI/HTTP context invoke `agh__tool_list`.
   - **Expected:** All registered tools with availability.
2. From a session-scoped invoker, invoke `agh__tool_list`.
   - **Expected:** Only the tools the session can call.
3. Invalid input (non-object) rejected with `schema_invalid`.

## Automation

- **Target:** Unit
- **Status:** Existing
- **Command/Spec:** `go test ./internal/tools -run TestNativeAghToolList`
