# TC-SEC-003: `agh__network_send` Rejects Raw Token Payloads And Metadata

**Priority:** P0 (Critical)
**Type:** Security / Redaction
**Status:** Not Run
**Estimated Time:** 15 minutes
**Created:** 2026-04-30
**Last Updated:** 2026-04-30

## Objective

Prove `agh__network_send` (and any AGH-owned message surface) rejects payloads or metadata that contain raw `claim_token` fields. Confirm `network_raw_token_rejected` is returned deterministically.

## Traceability

- Task: task_09.
- TechSpec: "Safety Invariants".
- ADR: ADR-005.
- Surfaces: `internal/tools/builtin/network.go`, network sender pipeline.

## Preconditions

- Isolated `AGH_HOME`.
- Two peer sessions in the workspace so a real network channel exists.

## Test Steps

1. **Well-formed message succeeds:**
   ```bash
   agh tool invoke agh__network_send --input '{
     "channel":"qa",
     "kind":"say",
     "body":{"hello":"world"}
   }' -o json | tee qa/logs/TC-SEC-003/send-good.json
   ```
   - **Expected:** Returns success; channel inbox shows the message.

2. **Body contains `claim_token`:**
   ```bash
   agh tool invoke agh__network_send --input '{
     "channel":"qa",
     "kind":"say",
     "body":{"claim_token":"agh_claim_abc"}
   }' -o json | tee qa/logs/TC-SEC-003/send-body-token.json
   ```
   - **Expected:** `error.reason_codes` includes `network_raw_token_rejected`. No message persisted.

3. **Extension metadata contains `claim_token`:**
   ```bash
   agh tool invoke agh__network_send --input '{
     "channel":"qa",
     "kind":"say",
     "body":{"hello":"world"},
     "ext":{"agh.metadata":{"claim_token":"agh_claim_xyz"}}
   }' -o json | tee qa/logs/TC-SEC-003/send-meta-token.json
   ```
   - **Expected:** Same `network_raw_token_rejected` reason.

4. **Nested body containing `claim_token` deep inside structured payload:**
   ```bash
   agh tool invoke agh__network_send --input '{
     "channel":"qa",
     "kind":"say",
     "body":{"a":{"b":{"claim_token":"agh_claim_deep"}}}
   }' -o json | tee qa/logs/TC-SEC-003/send-nested-token.json
   ```
   - **Expected:** Same rejection. Recursion-safe scrubber must catch nested fields.

5. **CLI parity (operator-side message):**
   ```bash
   agh network send --session sess-a --channel qa --kind say \
     --body '{"claim_token":"agh_claim_cli"}' -o json \
     | tee qa/logs/TC-SEC-003/cli-send-token.json
   ```
   - **Expected:** CLI returns the same deterministic error.

6. **Inbox grep:**
   ```bash
   agh network inbox --session sess-a -o json | tee qa/logs/TC-SEC-003/inbox.json
   grep -nE "claim_token" qa/logs/TC-SEC-003/inbox.json
   ```
   - **Expected:** Inbox has only the well-formed Step 1 message; no token-bearing payload made it through.

7. Run focused Go tests:
   ```bash
   go test ./internal/api/core ./internal/api/udsapi ./internal/cli ./internal/daemon \
     -run 'Test(NetworkConversionHelpersPreserveMetadata|NetworkHandlersValidateRequestsAndMapErrors|NetworkSendParsersRejectInvalidFlags|DaemonNativeTools)' \
     -count=1 | tee qa/logs/TC-SEC-003/network-send-tests.log
   ```

## Evidence To Capture

- All `qa/logs/TC-SEC-003/*.json` payloads.
- Inbox snapshot.
- Test logs.

## Edge Cases And Variations

| Variation | Input | Expected Result |
|-----------|-------|-----------------|
| Field name uses different case (`Claim_Token`) | mixed-case exact key | Rejected (the guard compares the `claim_token` key case-insensitively) |
| CamelCase non-canonical field (`claimToken`) | distinct key spelling | Allowed unless a future spec broadens the forbidden field-name set |
| Field value contains the substring "agh_claim_" | benign content like `"description":"see agh_claim_token docs"` | Allowed (the rule guards against the literal field name, not substring matches in arbitrary string values) — confirm the scrubber's exact policy from `internal/network` and align expectations. If the policy is broader, document accordingly. |
| `claim_token_hash` in body | observability metadata | Allowed; only `claim_token` is forbidden |
| Empty body | well-formed empty object | Allowed |

## Channels Exercised

- Tool invoke for `agh__network_send`.
- CLI `agh network send` if available.
- Persisted network inbox.

## Related Test Cases

- TC-SEC-001 (cross-channel raw-token redaction).
- TC-AUT-001 (autonomy flow).
