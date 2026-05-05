# TC-FUNC-011: Network Error Contract Mapping

**Priority:** P1
**Type:** Functional
**Module:** API Core
**Requirement:** Network endpoints must map validation, missing resource, and internal errors consistently.

## Objective

Verify `StatusForNetworkError`, create-channel status mapping, and endpoint-specific error responses.

## Preconditions

- API core can be exercised through handler tests.
- Network, workspace, and session error sentinels are available.

## Test Steps

1. Return `ErrNetworkValidation` from request conversion.
   **Expected:** Response status is 400.
2. Return `network.ErrTargetPeerNotFound` from send.
   **Expected:** Response status is 404.
3. Return `network.ErrInvalidField`.
   **Expected:** Response status is 400.
4. Return an unknown error from status or store access.
   **Expected:** Response status is 500.
5. Return workspace-not-found during create-channel.
   **Expected:** Response maps through workspace status rules.

## Edge Cases

| Variation | Input | Expected Result |
| --- | --- | --- |
| Wrapped sentinel | `fmt.Errorf("x: %w", sentinel)` | mapped by `errors.Is` |
| Decode error | malformed JSON body | 400 decode context |
| Network disabled | service endpoints | 503 except status endpoint |

## Related

- TC-FUNC-001
- TC-FUNC-008
- TC-FUNC-009
