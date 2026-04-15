## TC-PERF-002: Delivery Throughput Under Concurrent Load

**Priority:** P0
**Type:** Performance
**Status:** Not Run
**Estimated Time:** 20 minutes
**Created:** 2026-04-15

---

### Objective
Verify the Delivery Broker sustains high throughput under concurrent load across multiple route workers, maintains delivery ordering per route, keeps queue depths bounded, and drops no events.

### Preconditions
- [ ] Delivery Broker is initialized with 10 route workers and bounded queue size
- [ ] A mock delivery handler that records delivery order and timestamps is available
- [ ] Metrics collection for queue depth, latency, and throughput is instrumented
- [ ] Test environment has sufficient CPU cores (recommend 4+) for meaningful concurrency results
- [ ] No external dependencies (all delivery targets are in-process mocks)

### Performance Criteria
| Metric | Target | Acceptable | Actual | Status |
|--------|--------|------------|--------|--------|
| End-to-end delivery latency (p50) | < 10ms | < 25ms | | |
| End-to-end delivery latency (p95) | < 50ms | < 100ms | | |
| End-to-end delivery latency (p99) | < 100ms | < 250ms | | |
| Throughput (deliveries/second) | > 5,000 | > 2,000 | | |
| Events dropped | 0 | 0 | | |
| Delivery order violations per route | 0 | 0 | | |
| Max queue depth during test | < 2x queue capacity | < queue capacity | | |
| Goroutine count after drain | baseline + 0 | baseline + 0 | | |

### Load Scenarios

1. **Sustained concurrent enqueue**
   - Duration: 100 deliveries enqueued concurrently from 100 goroutines, distributed evenly across 10 route keys
   - Expected: All 100 deliveries complete successfully. Each route receives exactly 10 deliveries. Delivery order within each route matches enqueue order. No delivery is lost or duplicated.

2. **Burst traffic on a single route**
   - Duration: 200 deliveries enqueued in < 5ms, all targeting the same route key
   - Expected: The single route worker processes all 200 deliveries sequentially. Queue depth peaks but stays within configured bounds. No backpressure-induced drops. Total processing time scales linearly with delivery handler latency.

3. **Skewed route distribution**
   - Duration: 500 deliveries where 80% target 2 of 10 routes, 20% spread across the remaining 8
   - Expected: Hot routes show higher latency but no starvation on cold routes. All deliveries complete. Per-route ordering preserved. Total throughput degrades gracefully (no worse than 50% of uniform distribution).

4. **Worker drain and shutdown**
   - Duration: Enqueue 50 deliveries, then initiate graceful shutdown while deliveries are in-flight
   - Expected: All in-flight deliveries complete before shutdown returns. No deliveries are silently dropped. Shutdown completes within 5 seconds. All worker goroutines exit cleanly.

5. **Progressive sequencing validation**
   - Duration: Enqueue 100 deliveries per route across 10 routes, each delivery carrying a monotonic sequence number
   - Expected: The mock handler records sequence numbers in strictly increasing order per route. No gaps in sequence. Cross-route interleaving is permitted but intra-route order is absolute.

6. **Sustained load over time**
   - Duration: 30 seconds of continuous delivery at 200 deliveries/second across 10 routes
   - Expected: Throughput remains stable (no degradation > 10% from first 5s to last 5s). Queue depth does not grow monotonically. Memory footprint stabilizes within first 10 seconds. No goroutine leaks.

### Test Implementation Notes
- Use `sync.WaitGroup` to synchronize concurrent enqueue goroutines
- Record delivery timestamps with `time.Now()` at enqueue and completion for latency histograms
- Use `runtime.NumGoroutine()` before and after test to detect leaks
- Sort recorded deliveries by route key and verify ordering with sequence assertions
- For sustained load test, use a ticker-based producer with rate limiting

### Related Test Cases
- TC-PERF-001 (dedup cache must not bottleneck delivery path)
- TC-PERF-003 (batcher feeds into delivery broker)
- TC-PERF-005 (instance cache sync during delivery)
