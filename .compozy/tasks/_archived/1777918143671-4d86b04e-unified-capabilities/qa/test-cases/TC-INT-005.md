# TC-INT-005: Slack Delivery Acknowledgement Lifecycle

**Priority:** P1
**Type:** Integration
**Systems:** Slack Bridge, Slack API Client, Bridge Delivery Runtime
**API Endpoint:** Bridge delivery callback

## Objective

Verify outbound bridge deliveries create, update, delete, and resume Slack messages with correct ack state.

## Preconditions

- Slack API fake captures `chat.postMessage`, `chat.update`, and `chat.delete`.
- Delivery requests include ordered sequence values.
- Existing delivery state is available for update/delete flows.

## Test Steps

1. Deliver a start event with no remote message ID.
   **Expected:** Provider posts a new Slack message and returns ack with encoded remote message ID.
2. Deliver a final/edit event after start.
   **Expected:** Provider updates the existing message and advances sequence state.
3. Deliver a delete event.
   **Expected:** Provider deletes the remote message and returns ack preserving replace ID.
4. Deliver a resume event with snapshot.
   **Expected:** Provider restores state from snapshot and posts or updates according to remote message presence.
5. Deliver an out-of-order non-resume sequence.
   **Expected:** Provider rejects with out-of-order error.

## Edge Cases

| Variation | Input | Expected Result |
| --- | --- | --- |
| Missing remote ID on edit | no state or reference | error |
| Malformed remote ID | invalid encoded value | error |
| Slack API auth/rate/transient error | fake error status | classified recovery status |

## Related

- TC-SEC-004
