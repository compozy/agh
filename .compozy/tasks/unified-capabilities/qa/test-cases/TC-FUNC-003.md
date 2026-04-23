# TC-FUNC-003: API Peer Detail Metrics And Capability Catalog

**Priority:** P1
**Type:** Functional
**Module:** API Core
**Requirement:** Peer detail should expose correct identity, capabilities, and audit-derived metrics.

## Objective

Verify `GET /api/network/peers/{peer_id}` resolves the peer, loads appropriate audit entries, summarizes metrics, and returns rich capability catalog only when known.

## Preconditions

- Network service returns multiple peers across channels.
- Network store returns sent, received, delivered, and rejected audit rows.
- Session manager returns local session info.

## Test Steps

1. Request a present local peer by `peer_id`.
   **Expected:** Response is 200 and includes local session ID, display name, channel, peer card, capability catalog, and metrics.
2. Request a present remote peer by `peer_id`.
   **Expected:** Response is 200; metrics are filtered by channel and matching `peer_from` or `peer_to`.
3. Request an absent peer.
   **Expected:** Response is 404 and does not query audit for a nonexistent peer.
4. Return audit rows for other peers in the same channel.
   **Expected:** Metrics exclude unrelated rows.

## Edge Cases

| Variation | Input | Expected Result |
| --- | --- | --- |
| Blank peer ID | URL-encoded whitespace | 400 validation error |
| Audit store error | store returns error | 500 with store context |
| Capability catalog unknown | known flag false | detail omits rich catalog |

## Related

- TC-FUNC-002
- TC-PERF-003
