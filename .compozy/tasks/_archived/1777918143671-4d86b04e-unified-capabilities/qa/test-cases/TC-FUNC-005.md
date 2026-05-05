# TC-FUNC-005: API Channel Detail And History-Only Rooms

**Priority:** P1
**Type:** Functional
**Module:** API Core
**Requirement:** Channel detail should work for live and historical channels.

## Objective

Verify `GET /api/network/channels/{channel}` returns detail for active channels and history-only persisted channels while returning 404 for truly missing channels.

## Preconditions

- Channel name passes network channel validation.
- Store can return metadata and messages by channel.
- Runtime service can list peers by channel.

## Test Steps

1. Request an active channel with peers and sessions.
   **Expected:** Detail includes session payloads, peer payloads, counts, kind counts, purpose, and preview.
2. Request a history-only channel with messages but no active sessions or peers.
   **Expected:** Detail returns 200, `peer_count=0`, `session_count=0`, and preserved `message_count`.
3. Request a metadata-only channel with no messages.
   **Expected:** Detail returns 200 and uses metadata timestamps for last activity.
4. Request a missing channel.
   **Expected:** 404 channel not found.

## Edge Cases

| Variation | Input | Expected Result |
| --- | --- | --- |
| Invalid channel | channel violates `ValidateChannel` | 400 validation error |
| Session list failure | `Sessions.ListAll` error | 500 |
| Cursor-specific message error | store returns cursor missing | detail request does not use cursor |

## Related

- TC-FUNC-004
- TC-UI-005
