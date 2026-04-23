# TC-PERF-001: Channel Aggregation Query Volume

**Priority:** P1
**Type:** Performance
**Module:** API Core + Global DB
**Requirement:** Channel list aggregation must avoid per-channel N+1 store or session calls.

## Objective

Verify `GET /api/network/channels` loads sessions, peers, metadata, and messages with fixed query/call counts as channel count grows.

## Preconditions

- Fixture can create hundreds of channels and messages.
- Store and network/session stubs can count calls.
- API handler exposes channel list endpoint.

## Test Steps

1. Seed 100 channels with metadata and timeline rows.
   **Expected:** Store seed succeeds within local test time budget.
2. Request channel list once.
   **Expected:** Handler performs one session list, one peer list, one channel-metadata list, and one message list.
3. Measure response size and time in test/benchmark mode.
   **Expected:** Response remains deterministic and within project benchmark expectations.
4. Add history-only channels.
   **Expected:** Query volume remains fixed and summaries include history-only channels.

## Edge Cases

| Variation | Input | Expected Result |
| --- | --- | --- |
| 0 channels | empty store | empty list quickly |
| 500+ messages | high timeline count | aggregation remains bounded by current query policy |
| Same timestamp messages | tie-breaker by message ID | deterministic preview and sort |

## Related

- TC-FUNC-004
- TC-PERF-002
