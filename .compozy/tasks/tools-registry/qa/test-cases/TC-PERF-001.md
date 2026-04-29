# TC-PERF-001 — Concurrent dispatch into the same tool preserves hook order and result limiter integrity

- **Priority:** P1
- **Type:** Performance / concurrency
- **Trace:** Task 04, AGENTS.md concurrency discipline

## Objective

Prove that 50 concurrent invocations of the same tool produce 50 independent dispatch traces with correct hook ordering, no shared mutable state corruption, no truncation crosstalk, and no race detector warnings.

## Test Steps

1. Configure a tool whose handler returns a 1MB payload over `max_result_bytes = 256KB`.
2. Launch 50 concurrent calls via UDS; collect results.
   - **Expected:** Each result is independently truncated; `bytes` field reflects the original size; `truncated = true` for each.
3. Configure pre-call hook with bounded latency.
   - **Expected:** Hook ordering preserved per-call; no cross-call state.
4. Run with `-race`.
   - **Expected:** No race detected.

## Automation

- **Target:** Integration
- **Status:** Missing systematic concurrency stress; Existing baseline race tests
- **Command/Spec:** `go test ./internal/tools -run TestDispatchConcurrencyStress -race`
