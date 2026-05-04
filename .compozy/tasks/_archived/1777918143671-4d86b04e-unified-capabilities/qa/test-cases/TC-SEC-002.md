# TC-SEC-002: Slack Webhook Guard Limits

**Priority:** P1
**Type:** Security
**Module:** Slack Bridge
**Requirement:** Webhook guard must constrain request method, content type, body size, rate, and concurrency.

## Objective

Verify Slack webhook handling applies guardrails before dispatching payloads into bridge host APIs.

## Preconditions

- Slack provider route is active.
- Route has fixed-window rate limiter and in-flight limiter.
- Test client can vary method, content type, body size, and parallelism.

## Test Steps

1. Send a GET request to the webhook route.
   **Expected:** Request is rejected by allowed-method guard and no ingest occurs.
2. Send POST with unsupported content type.
   **Expected:** Request is rejected by content-type guard.
3. Send POST with a body over 1 MiB.
   **Expected:** Request is rejected by body-size guard.
4. Send requests above rate limit for the same remote address and instance.
   **Expected:** Excess requests are rejected and no ingest occurs for rejected requests.
5. Saturate the in-flight limiter.
   **Expected:** Excess concurrent requests are rejected or delayed per guard semantics without data races.

## Edge Cases

| Variation | Input | Expected Result |
| --- | --- | --- |
| Unknown path | `/missing` | 404 |
| Conflicting webhook path | two routes same path | both degraded and no route selected |
| Allowed form content type | `application/x-www-form-urlencoded` | accepted when signed |

## Related

- TC-INT-103
- TC-INT-104
