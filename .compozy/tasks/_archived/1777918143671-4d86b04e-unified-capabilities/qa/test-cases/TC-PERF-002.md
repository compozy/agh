# TC-PERF-002: Timeline Pagination And Index Usage

**Priority:** P1
**Type:** Performance
**Module:** Global DB
**Requirement:** Message timeline reads must stay bounded under large logs.

## Objective

Verify channel and peer timeline queries use indexed filters, limits, cursor lookups, and deterministic ordering under load.

## Preconditions

- Global DB contains thousands of `network_timeline_log` rows across channels, peers, kinds, and timestamps.
- Tests can run with `-race` and package benchmarks when needed.

## Test Steps

1. Query recent channel messages with a limit.
   **Expected:** Query returns only the limit and respects `(channel, timestamp, message_id)` order.
2. Query peer directed messages with a limit.
   **Expected:** Query filters by peer and `DirectedOnly` without scanning unrelated undirected messages in result.
3. Query before and after cursors repeatedly.
   **Expected:** Pages do not duplicate or skip rows.
4. Run relevant benchmark or targeted test.
   **Expected:** No unexpected regression from Round 1 timing baseline.

## Edge Cases

| Variation | Input | Expected Result |
| --- | --- | --- |
| Cursor at first row | `before=first` | empty page |
| Cursor at last row | `after=last` | empty page |
| Same timestamp burst | 100 rows same timestamp | stable message ID tie-break |

## Related

- TC-FUNC-006
- TC-FUNC-007
- TC-SEC-003
