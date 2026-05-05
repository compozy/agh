# TC-FUNC-007: Peer Message Directed Timeline

**Priority:** P1
**Type:** Functional
**Module:** Store + API Core
**Requirement:** Peer timelines only include directed messages involving the selected peer.

## Objective

Verify `GET /api/network/peers/{peer_id}/messages` validates the peer, applies `DirectedOnly`, filters by either `peer_from` or `peer_to`, and preserves local author session metadata.

## Preconditions

- Network service returns a selected peer plus at least one local peer in the same channel.
- Store contains broadcast and directed messages.
- Some directed messages target other peers.

## Test Steps

1. Request messages for an existing remote peer.
   **Expected:** API queries store with `PeerID=<peer>` and `DirectedOnly=true`.
2. Store returns directed messages where the peer is sender and recipient.
   **Expected:** Response includes both directions.
3. Store contains broadcast `say` messages in same channel.
   **Expected:** Broadcast messages are excluded.
4. Request messages for a missing peer.
   **Expected:** API returns 404 before querying store messages.

## Edge Cases

| Variation | Input | Expected Result |
| --- | --- | --- |
| Cursor from other peer | `before` or `after` | 400 cursor not found |
| Local sent message | `direction=sent` and session ID present | response has `local=true` and session ID |
| Remote received message | no session ID | response keeps remote peer display name |

## Related

- SMOKE-003
- TC-FUNC-003
