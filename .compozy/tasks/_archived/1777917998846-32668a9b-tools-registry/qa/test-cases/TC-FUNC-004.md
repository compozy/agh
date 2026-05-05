# TC-FUNC-004 — `id_too_long` reason code on over-length canonical IDs

- **Priority:** P1
- **Type:** Functional / boundary
- **Trace:** Task 01, ADR-007

## Objective

Prove that any sanitized external MCP/extension name that produces a canonical ID > 64 chars is rejected with reason `id_too_long`; AGH never truncates or hash-suffixes.

## Test Steps

1. Configure an MCP server whose name is exactly 60 chars, with a tool whose name ensures sanitization > 64.
   - **Expected:** Tool descriptor present operator-visible with `id_too_long`; session-hidden.
2. Try registering a `native_go` tool with a 65-char canonical id.
   - **Expected:** Registration error.
3. Confirm operator diagnostics carry the exact `id_too_long` reason and that error text never includes a fabricated truncated ID.

## Automation

- **Target:** Unit
- **Status:** Existing
- **Command/Spec:** `go test ./internal/tools -run TestIDTooLong`
