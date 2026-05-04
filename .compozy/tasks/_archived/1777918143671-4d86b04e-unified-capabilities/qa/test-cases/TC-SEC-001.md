# TC-SEC-001: Slack Signature Verification

**Priority:** P0
**Type:** Security
**Module:** Slack Bridge
**Requirement:** Slack webhooks must reject unsigned, stale, or tampered requests.

## Objective

Verify Slack signature validation enforces signing secret presence, timestamp freshness, HMAC match, and request/body integrity.

## Preconditions

- Slack provider route has a configured signing secret.
- Test helper can generate Slack `v0` signatures.
- Webhook body can be replayed with modified headers.

## Test Steps

1. Send a valid signed request.
   **Expected:** Signature verification succeeds and the request reaches the Slack webhook mapper.
2. Send the same body with an invalid signature.
   **Expected:** HTTP 401 and no ingest.
3. Send a valid signature with stale timestamp.
   **Expected:** HTTP 401 and no ingest.
4. Send missing signature headers.
   **Expected:** HTTP 401 and no ingest.
5. Configure empty signing secret.
   **Expected:** Provider marks instance auth required or rejects verification; unsigned requests are never accepted.

## Edge Cases

| Variation | Input | Expected Result |
| --- | --- | --- |
| Invalid timestamp | non-integer header | verification error |
| Nil request | direct function call | validation error |
| Body tamper | one-byte body change | HMAC mismatch |

## Related

- TC-INT-103
- TC-SEC-002
