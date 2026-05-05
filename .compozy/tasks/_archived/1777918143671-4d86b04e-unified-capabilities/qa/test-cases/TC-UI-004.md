# TC-UI-004: Network Composer Send States

**Priority:** P1
**Type:** UI
**Module:** Web Network Workspace
**Requirement:** Composer should send through an eligible local session and handle disabled states.

## Objective

Verify broadcast and direct composing builds the correct `NetworkSendRequest`, disables when no local session is available, and resets state after success.

## Preconditions

- Network workspace has one channel room and one peer room.
- Channel detail includes at least one local session for enabled send tests.
- Direct peer has a local peer in its channel for enabled direct tests.

## Test Steps

1. Select a channel with a local session.
   **Expected:** Composer is enabled with broadcast placeholder.
2. Type a message and click Send.
   **Expected:** Mutation body uses `kind="say"`, channel ID, first local session ID, and JSON text body.
3. Select a remote peer with a local sender in the same channel.
   **Expected:** Composer is enabled with direct placeholder.
4. Type a direct message and click Send.
   **Expected:** Mutation body uses `kind="direct"`, target peer ID, channel, local session ID, and text body.
5. Select a room without local sender.
   **Expected:** Composer is disabled or submit shows no-local-session error.

## Edge Cases

| Variation | Input | Expected Result |
| --- | --- | --- |
| Whitespace draft | `"   "` | Send disabled |
| Mutation failure | API error | toast error and draft preserved |
| Mutation success | API OK | draft clears and network queries invalidate |

## Related

- SMOKE-003
- TC-FUNC-009
