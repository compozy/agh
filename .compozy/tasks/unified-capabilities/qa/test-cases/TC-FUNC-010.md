# TC-FUNC-010: Network Inbox Query Semantics

**Priority:** P1
**Type:** Functional
**Module:** API Core + Network Runtime
**Requirement:** Inbox reads must be explicit and preserve full envelope payloads.

## Objective

Verify `GET /api/network/inbox` validates the target session, accepts the legacy `session` alias, and serializes queued envelopes without losing proof or extension metadata.

## Preconditions

- Network is enabled.
- Runtime inbox service returns queued envelopes.
- Envelopes include proof, ext, expires_at, reply_to, trace_id, and causation_id variants.

## Test Steps

1. Request inbox with `session_id=sess-a`.
   **Expected:** Runtime receives `sess-a`; response includes queued envelopes.
2. Request inbox with `session=sess-a`.
   **Expected:** Alias is accepted and maps to same runtime call.
3. Request inbox without either query parameter.
   **Expected:** 400 `session_id query is required`.
4. Runtime returns invalid field.
   **Expected:** 400 mapped network error.

## Edge Cases

| Variation | Input | Expected Result |
| --- | --- | --- |
| Empty queue | no envelopes | `messages: []` |
| Proof metadata | non-empty proof map | cloned and serialized |
| Expiring envelope | `expires_at` set | response keeps unix seconds |

## Related

- SMOKE-003
- TC-INT-006
