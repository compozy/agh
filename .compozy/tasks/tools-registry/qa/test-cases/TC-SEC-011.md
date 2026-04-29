# TC-SEC-011 — Approval token is single-use and bound to tool/session/workspace/input

- **Priority:** P0
- **Type:** Security / approval lifecycle
- **Trace:** Task 11, ADR-005, Safety Invariant 27

## Objective

Prove `POST /api/tools/{id}/approvals` issues a single-use approval token bound to `tool_id`, `session_id`, `workspace_id`, and `input_digest`. Replay, mismatch, expiration, or missing-token cases return deterministic reason codes.

## Preconditions

- `permissions.mode = "approve-reads"` for a mutating tool path requiring approval.
- One mutating extension-host tool requiring approval (`ext__test__write_thing`).

## Test Steps

1. `POST /api/tools/ext__test__write_thing/approvals` with `{session_id, workspace_id, input}`.
   - **Expected:** Response includes `approval_token` (`APPROVAL_TOKEN_v1_TESTONLY`), `expires_at`, `tool_id`, `input_digest`.
2. `POST /api/tools/ext__test__write_thing/invoke` with the token and matching input.
   - **Expected:** `200`, call succeeds.
3. Re-replay the same token.
   - **Expected:** `403` / `tool_denied` with reason `approval_token_replayed`.
4. Mint another token; invoke with a different input value (different `input_digest`).
   - **Expected:** `403` reason `approval_token_mismatch`.
5. Mint another token; wait beyond `[tools.policy].approval_timeout_seconds`.
   - **Expected:** `403` reason `approval_token_expired`.
6. Invoke without supplying any token.
   - **Expected:** `403` reason `approval_token_missing` OR (depending on mode) `approval_required`.
7. Mint a token, restart daemon, then invoke.
   - **Expected:** `403` reason `approval_token_expired` because tokens live only in daemon memory.

## Edge Cases

- Token must be hashed in storage; token raw value MUST NOT appear in logs/events/SSE/persisted state (covered by TC-SEC-013).
- CLI parity: `agh tool approve <id> --session <sid> --workspace <wsid> --input <json> -o json` produces equivalent payload.

## Automation

- **Target:** Integration
- **Status:** Existing
- **Command/Spec:** `go test ./internal/api/core -run TestApprovalTokenLifecycle`
