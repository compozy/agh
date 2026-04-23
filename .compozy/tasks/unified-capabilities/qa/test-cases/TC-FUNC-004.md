# TC-FUNC-004: API Channel Summary Aggregation

**Priority:** P0
**Type:** Functional
**Module:** API Core
**Requirement:** Channel summaries must aggregate metadata, sessions, peers, and messages consistently.

## Objective

Verify `GET /api/network/channels` returns a complete operator-facing channel list using durable metadata, active sessions, runtime peers, and persisted message history.

## Preconditions

- Network store returns channel metadata and timeline entries.
- Session manager returns active and stopped sessions.
- Network service returns local and remote peers.

## Test Steps

1. Include a channel with metadata, active sessions, peers, and messages.
   **Expected:** Summary includes workspace ID, purpose, created-by, created-at, peer counts, session count, message count, last activity, and preview.
2. Include a history-only channel with messages but no sessions or peers.
   **Expected:** Summary is still returned with `message_count > 0`.
3. Include a stopped session in a channel.
   **Expected:** Stopped session does not count as active, but its persisted history can keep the channel visible.
4. Include two channels with different activity timestamps.
   **Expected:** Sort order is latest activity descending, then channel name ascending.

## Edge Cases

| Variation | Input | Expected Result |
| --- | --- | --- |
| Metadata-only channel | no sessions, peers, or messages | visible with metadata fields |
| Empty store/runtime | no channels | empty list, not null |
| Store query failure | messages or metadata fail | 500 with wrapped store error |

## Related

- SMOKE-002
- TC-FUNC-005
- TC-PERF-001
