# TC-INT-102: Network Runtime Send Receive Audit Timeline

**Priority:** P0
**Type:** Integration
**Systems:** Network Manager, Router, Peer Registry, Delivery Coordinator, Global DB
**API Endpoint:** `/api/network/send`

## Objective

Verify local runtime peer operations publish, receive, audit, persist, deliver, and shut down cleanly under race-enabled tests.

## Preconditions

- Network manager can be built with local test transport.
- Audit writer is connected to a Global DB store.
- Prompt delivery fixture can accept network prompts.

## Test Steps

1. Join two local sessions to the same channel.
   **Expected:** Peer registry lists both peers and channel count is one.
2. Send a broadcast `say`.
   **Expected:** Audit records sent and received events; timeline stores one sent/received message per durable semantics.
3. Send a directed `direct` message.
   **Expected:** Only the target session receives delivery; peer-directed timeline includes the message.
4. Deliver queued message after prompt turn ends.
   **Expected:** Delivered metric increments and no message remains stuck in inbox.
5. Shut down the manager under race test.
   **Expected:** Heartbeats, subscriptions, and delivery workers exit without race or leak symptoms.

## Edge Cases

| Variation | Input | Expected Result |
| --- | --- | --- |
| Duplicate inbound envelope | same message ID within replay window | rejected duplicate receipt where applicable |
| Target not present | directed send to absent peer | target-not-found error |
| Queue overflow | depth exceeded | oldest item dropped and warning logged |

## Related

- SMOKE-003
- TC-PERF-002
