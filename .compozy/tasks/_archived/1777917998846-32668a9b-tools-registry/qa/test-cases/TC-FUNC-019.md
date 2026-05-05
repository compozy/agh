# TC-FUNC-019 — `agh__tool_search` returns hits filtered by id/title/desc/source/tags/toolsets

- **Priority:** P2
- **Type:** Functional / native tool
- **Trace:** Task 05, ADR-004

## Test Steps

1. Search by canonical id substring (`skill`).
   - **Expected:** All `agh__skill_*` tools returned.
2. Search by source.kind (`mcp`).
   - **Expected:** Only MCP-backed tools returned.
3. Search with empty query.
   - **Expected:** Returns all (or rejects per descriptor).
4. Search exceeding the result budget triggers truncation metadata.

## Automation

- **Target:** Unit
- **Status:** Existing
- **Command/Spec:** `go test ./internal/tools -run TestNativeAghToolSearch`
