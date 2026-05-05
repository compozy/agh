# TC-FUNC-031 — Result limiter truncates oversized output identically across CLI/HTTP/UDS/MCP

- **Priority:** P0
- **Type:** Functional / result budget
- **Trace:** Task 04, Safety Invariant 11

## Objective

Prove the central result limiter applies before results cross any surface. Truncation behavior is identical across CLI, HTTP, UDS, hosted MCP, and SSE/event payloads for the same call.

## Test Steps

1. Tool descriptor `max_result_bytes = 1024`. Provider returns 4096 bytes.
2. Invoke via HTTP, UDS, CLI, hosted MCP `tools/call`.
   - **Expected:** All four return `truncated = true`, `bytes = 4096` (recorded), and identical truncated content/preview/artifacts contract.
3. Inspect SSE/event payload — same truncation metadata.
4. Reduce descriptor budget to 0 (invalid).
   - **Expected:** Config validation rejects.

## Automation

- **Target:** Integration
- **Status:** Existing partial
- **Command/Spec:** `go test ./internal/tools -run TestResultBudgetParity`
