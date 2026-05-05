# TC-FUNC-003 — `Canonicalize(rawServer, rawTool)` for MCP sources is byte-stable

- **Priority:** P1
- **Type:** Functional / sanitization
- **Trace:** Task 09, ADR-010

## Objective

Prove the shared `Canonicalize(rawServer, rawTool)` helper produces deterministic results across `MCPCallExecutor`, MCP descriptor normalization, hosted MCP registration, and config/resource validation; matches the shared fixture set.

## Test Steps

1. Run shared canonical fixtures (`internal/extension/testdata/digest/cases.json`, `sdk/typescript/test-fixtures/digest/cases.json`, `sdk/go/test-fixtures/digest/cases.json` and the canonicalize fixture set).
   - **Expected:** All implementations agree byte-for-byte.
2. Pass raw names with: leading/trailing whitespace, internal `-` and `.`, mixed case, leading digit, ASCII control chars.
   - **Expected:** Whitespace trimmed; `-`/`.` mapped to `_`; lowercased; leading-digit segments rejected.
3. Pass raw names whose normalized form would collide with another already-registered tool.
   - **Expected:** Both kept operator-visible with `conflicted_sanitized_name`; session projection hides them.
4. Pass over-length combinations.
   - **Expected:** `id_too_long`; AGH does not truncate.

## Automation

- **Target:** Unit
- **Status:** Existing
- **Command/Spec:** `go test ./internal/tools -run TestCanonicalize`
