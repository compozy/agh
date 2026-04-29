# TC-SEC-013 — Approval token absent from logs/events/SSE/hosted MCP/diagnostics

- **Priority:** P0
- **Type:** Security / redaction
- **Trace:** Task 11, ADR-005, Safety Invariant 27

## Objective

Prove the raw approval token never appears in logs, events, SSE payloads, hosted MCP responses, persisted state, or operator diagnostics. Only the authenticated issuance response and the matching invoke request may carry the raw value.

## Preconditions

- Sentinels: approval token returned from issuance is `APPROVAL_TOKEN_v1_TESTONLY`.

## Test Steps

1. `POST /api/tools/{id}/approvals` from CLI/HTTP client.
   - **Expected:** Token returned in body; capture exact value `APPROVAL_TOKEN_v1_TESTONLY`.
2. `POST /api/tools/{id}/invoke` with that token.
   - **Expected:** Successful invoke.
3. Capture daemon logs, event journal, SSE stream, hosted MCP responses, settings output, web UI payload, persisted DB inspection.
4. Run sentinel scan.
   - **Expected:** Zero matches outside the issuance response and the matching invoke request capture.
5. Inspect `tool.call_started`, `tool.call_completed`, `tool.policy_evaluated` events.
   - **Expected:** Each event includes hashed/redacted token reference but never the raw token.

## Edge Cases

- CLI `agh tool invoke ... --approval-token-file` reads the token without printing it on stdout/stderr.
- Audit/replay viewer must show only the hash/identifier when reviewing call history.

## Automation

- **Target:** Integration
- **Status:** Existing partial; Missing systematic sentinel scan
- **Command/Spec:** `go test ./internal/observe -run TestApprovalTokenRedaction`
