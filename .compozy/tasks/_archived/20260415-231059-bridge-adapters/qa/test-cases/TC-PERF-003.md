## TC-PERF-003: Inbound Batching Efficiency

**Priority:** P1
**Type:** Performance
**Status:** Not Run
**Estimated Time:** 15 minutes
**Created:** 2026-04-15

---

### Objective

Verify the InboundBatcher coalesces rapid-fire messages into minimal batches while preserving message ordering, respects debounce windows, and dispatches batches within acceptable latency bounds.

### Preconditions

- [ ] InboundBatcher is configured with 25ms debounce window
- [ ] Configurable split thresholds are set to test defaults (e.g., max batch size = 100)
- [ ] A mock batch consumer that records batch contents, sizes, and timestamps is available
- [ ] System clock resolution is sufficient for sub-millisecond timing (use `time.Now()` with monotonic clock)
- [ ] No other batcher instances are running in the test process

### Performance Criteria

| Metric                                         | Target    | Acceptable | Actual | Status |
| ---------------------------------------------- | --------- | ---------- | ------ | ------ |
| Messages coalesced per batch (50 msgs in 25ms) | >= 40     | >= 25      |        |        |
| Total batch count for 50 rapid messages        | <= 2      | <= 3       |        |        |
| Flush latency (first msg to batch dispatch)    | < 50ms    | < 100ms    |        |        |
| Message ordering within batch                  | preserved | preserved  |        |        |
| Message loss                                   | 0         | 0          |        |        |
| Batch dispatch overhead per batch              | < 1ms     | < 5ms      |        |        |
| Split threshold enforcement                    | exact     | exact      |        |        |

### Load Scenarios

1. **Rapid burst coalescing**
   - Duration: Send 50 messages within 5ms for the same routing key
   - Expected: All 50 messages are coalesced into 1 batch (or at most 2 if split threshold triggers). Messages within the batch are in send order. Batch is dispatched within 25ms + debounce window after the last message. No messages are lost.

2. **Debounce window reset**
   - Duration: Send 10 messages at t=0ms, then 10 more at t=20ms (within the 25ms debounce window)
   - Expected: All 20 messages are coalesced into a single batch because the second burst resets the debounce timer. Batch dispatches ~25ms after the last message (t=45ms). Message order is [first 10, then second 10].

3. **Debounce window expiry between bursts**
   - Duration: Send 10 messages at t=0ms, wait 50ms (debounce expires), then send 10 more messages
   - Expected: Two separate batches are dispatched. First batch contains the first 10 messages, second batch contains the next 10. Each batch preserves internal ordering.

4. **Split threshold enforcement**
   - Duration: Send 150 messages rapidly for the same routing key with max batch size = 100
   - Expected: At least 2 batches are produced. First batch contains exactly 100 messages. Second batch contains the remaining 50. Message ordering is preserved across the split boundary (batch 1 messages all precede batch 2 messages).

5. **Multi-key isolation**
   - Duration: Send 30 messages for key A and 30 messages for key B, interleaved in rapid succession
   - Expected: Key A messages are batched separately from key B messages. Each key's batch preserves its own ordering. No cross-contamination of messages between keys.

6. **Latency measurement under load**
   - Duration: Send 500 messages across 10 routing keys over 1 second (50 per key, spread over 100ms bursts)
   - Expected: p50 end-to-end latency (from send to batch dispatch) is < 50ms. p99 is < 100ms. All 500 messages are accounted for in dispatched batches. No batch contains messages from multiple routing keys.

### Test Implementation Notes

- Timestamp each message at send time and each batch at dispatch time for latency calculation
- Use channels or mutex-protected slices in the mock consumer to safely record batches from concurrent dispatches
- Verify ordering by embedding a monotonic sequence number in each message payload
- For debounce window tests, use `time.Sleep` between bursts (acceptable here since we are testing timer behavior)
- Run with `-race` flag to catch any concurrent access issues in the batcher

### Related Test Cases

- TC-PERF-002 (batcher output feeds into delivery broker)
- TC-PERF-004 (rate limiter may interact with batched deliveries)
