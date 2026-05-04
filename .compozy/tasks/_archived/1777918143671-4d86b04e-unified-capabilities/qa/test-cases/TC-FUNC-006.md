# TC-FUNC-006: Channel Message Timeline Cursor Semantics

**Priority:** P1
**Type:** Functional
**Module:** Store + API Core
**Requirement:** Channel timelines must paginate without cross-channel leakage.

## Objective

Verify `GET /api/network/channels/{channel}/messages` and `ListNetworkMessages` honor channel scoping, limits, before/after cursors, ordering, and missing cursor handling.

## Preconditions

- Global DB has messages in at least two channels.
- Multiple messages share timestamps to exercise message-ID tie-breakers.
- API handler has sessions and peers for display-name enrichment.

## Test Steps

1. Query channel messages with no cursor.
   **Expected:** Messages are returned ascending by timestamp and message ID for the requested channel only.
2. Query `before=<message_id>` with a limit.
   **Expected:** Returns the immediately previous page in ascending display order.
3. Query `after=<message_id>` with a limit.
   **Expected:** Returns subsequent messages in ascending order.
4. Use a cursor ID from another channel.
   **Expected:** Store returns `sql.ErrNoRows`; API maps it to 400 `message cursor not found`.
5. Request messages for a missing channel with no messages, sessions, peers, or metadata.
   **Expected:** 404 channel not found.

## Edge Cases

| Variation | Input | Expected Result |
| --- | --- | --- |
| Both cursors | `before` and `after` | 400 validation error |
| Negative limit | `limit=-1` | 400 validation error |
| Remote author | received message | display name from remote peer when visible |

## Related

- SMOKE-003
- TC-PERF-002
- TC-SEC-003
