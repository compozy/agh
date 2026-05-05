# TC-SEC-003: Network API Injection And Boundary Inputs

**Priority:** P1
**Type:** Security
**Module:** API Core + Global DB
**Requirement:** Network filters and cursors must use parameterized queries and validation.

## Objective

Verify channel, peer, kind, direction, message ID, cursor, and limit inputs cannot alter SQL semantics or bypass endpoint scoping.

## Preconditions

- Global DB has multiple channels, peers, and messages.
- API handlers expose channel and peer message endpoints.
- Inputs include SQL-like strings, whitespace, and invalid channel characters.

## Test Steps

1. Query channel messages with channel containing SQL-like punctuation.
   **Expected:** Channel validation rejects invalid names or parameterized query returns no rows without leakage.
2. Query peer messages with peer ID containing SQL-like punctuation.
   **Expected:** Peer lookup requires exact peer ID match and returns 404 if absent.
3. Query cursor from another channel or peer.
   **Expected:** 400 cursor-not-found response; no cross-scope rows leak.
4. Query negative limit.
   **Expected:** 400 validation error before SQL execution.
5. Query very large limit.
   **Expected:** Repository limit policy is enforced or query remains bounded by accepted contract.

## Edge Cases

| Variation | Input | Expected Result |
| --- | --- | --- |
| Unicode channel | non-ASCII channel | validation-defined accept/reject behavior |
| Empty peer ID | whitespace path | 400 |
| Both cursors | `before` and `after` | 400 |

## Related

- TC-FUNC-006
- TC-FUNC-007
- TC-PERF-002
