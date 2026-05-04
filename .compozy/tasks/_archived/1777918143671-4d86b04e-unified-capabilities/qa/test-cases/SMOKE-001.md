# SMOKE-001: Network Status And Peer Listing

**Priority:** P0
**Type:** Smoke
**Status:** Not Run
**Estimated Time:** 10 minutes
**Created:** 2026-04-23

## Objective

Verify the minimum network operator path: status can be read, peer listing responds, and disabled mode reports a safe status instead of failing.

## Preconditions

- Repository dependencies are installed.
- API handlers can be exercised through tests or a local daemon.
- Network config can be toggled enabled and disabled in test fixtures.

## Test Steps

1. Request network status with network disabled.
   **Expected:** Response is 200 with `enabled=false` and `status="disabled"`.
2. Request network status with network enabled and a runtime status fixture.
   **Expected:** Response is 200 and includes listener, peer, channel, queue, delivery, and kind metrics.
3. Request network peers without a channel filter.
   **Expected:** Response is 200 and returns local and remote peers with display names and peer cards.
4. Request network peers with `channel=builders`.
   **Expected:** Response only includes peers visible in `builders`.

## Edge Cases

| Variation | Input | Expected Result |
| --- | --- | --- |
| Missing network service while enabled | `GET /api/network/peers` | 503 service unavailable |
| Session enrichment fails | `Sessions.ListAll` error | Peer list still returns best-effort payload |
| Blank channel filter | `channel=%20` | Treated as no filter |

## Related

- TC-FUNC-001
- TC-FUNC-002
- TC-INT-006
