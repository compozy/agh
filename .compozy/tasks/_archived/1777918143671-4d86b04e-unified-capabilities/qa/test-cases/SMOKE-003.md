# SMOKE-003: Send Message And Timeline Visibility

**Priority:** P0
**Type:** Smoke
**Status:** Not Run
**Estimated Time:** 15 minutes
**Created:** 2026-04-23

## Objective

Verify that a network message can be sent through a local session and appears in the persisted channel or peer timeline.

## Preconditions

- Network is enabled.
- At least one local peer is joined to a channel.
- Network audit writer is connected to a message-capable store.

## Test Steps

1. Send a `say` payload with `session_id`, `channel`, JSON body, and extension metadata.
   **Expected:** API returns 200 with assigned message ID and preserves metadata.
2. Query channel messages for the channel.
   **Expected:** Timeline contains the sent message with text, preview, direction, local session ID, and UTC timestamp.
3. Send a `direct` message to a visible peer.
   **Expected:** API returns 200 and peer messages include only directed entries involving that peer.
4. Query inbox for the local session after inbound delivery is queued.
   **Expected:** Inbox returns queued envelopes with protocol metadata and body intact.

## Edge Cases

| Variation | Input | Expected Result |
| --- | --- | --- |
| Invalid JSON body | malformed body | 400 validation error |
| Missing target peer | direct `to` not present | 404 target peer not found |
| Duplicate message ID | same message ID written twice | timeline keeps one row |

## Related

- TC-FUNC-006
- TC-FUNC-007
- TC-FUNC-009
- TC-INT-102
