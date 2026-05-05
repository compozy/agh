# TC-PERF-003 — Approval-token issuance + invoke under load

- **Priority:** P2
- **Type:** Performance / concurrency
- **Trace:** Task 11, ADR-005, Safety Invariant 27

## Objective

Prove approval-token issuance and consumption preserve single-use semantics under concurrent issuance/invoke combinations.

## Test Steps

1. 20 concurrent `POST /api/tools/{id}/approvals` calls with the same `(tool_id, session_id, workspace_id, input)`.
   - **Expected:** Each issues a unique token; no shared token; daemon-memory storage is hash-only.
2. Replay all 20 tokens concurrently against `/invoke`.
   - **Expected:** Each token consumed exactly once; replays of any token return `approval_token_replayed`.
3. Mismatched input on one of the invokes → `approval_token_mismatch`.
4. Run with `-race`.

## Automation

- **Target:** Integration
- **Status:** Missing
- **Command/Spec:** `go test ./internal/api/core -run TestApprovalConcurrency -race`
