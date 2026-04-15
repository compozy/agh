## TC-PERF-004: Rate Limiter Fairness Under Multi-Key Load

**Priority:** P1
**Type:** Performance
**Status:** Not Run
**Estimated Time:** 15 minutes
**Created:** 2026-04-15

---

### Objective

Verify the FixedWindowRateLimiter enforces per-key rate limits accurately, distributes capacity fairly across keys, does not starve any key under contention, and maintains consistent enforcement as windows rotate.

### Preconditions

- [ ] FixedWindowRateLimiter is configured with a known window duration (e.g., 100ms for test speed)
- [ ] Rate limit per key is set to a measurable value (e.g., 10 requests per window)
- [ ] Test harness can record per-key accept/reject decisions with timestamps
- [ ] System clock is reliable for window boundary detection
- [ ] No other rate limiter instances sharing state in the test process

### Performance Criteria

| Metric                                                       | Target             | Acceptable         | Actual | Status |
| ------------------------------------------------------------ | ------------------ | ------------------ | ------ | ------ |
| Per-key acceptance accuracy                                  | 100% correct       | 100% correct       |        |        |
| Per-key acceptance count per window                          | exactly limit      | exactly limit      |        |        |
| Per-key rejection count (over-limit)                         | remaining requests | remaining requests |        |        |
| Cross-key fairness (max deviation from mean acceptance rate) | < 5%               | < 10%              |        |        |
| Keys starved (0 acceptances in any window)                   | 0                  | 0                  |        |        |
| Window rotation latency (counter reset)                      | < 1ms              | < 5ms              |        |        |
| Concurrent enforcement accuracy (8 goroutines)               | 100% correct       | 100% correct       |        |        |
| Rate limiter lookup throughput (ops/sec)                     | > 1,000,000        | > 500,000          |        |        |

### Load Scenarios

1. **Uniform distribution across keys**
   - Duration: Send 500 requests distributed evenly across 50 keys (10 per key) within one rate limit window (100ms), with limit = 10 per key per window
   - Expected: All 500 requests are accepted. Each key receives exactly 10 acceptances. Zero rejections.

2. **Over-limit enforcement per key**
   - Duration: Send 20 requests for a single key within one window, with limit = 10
   - Expected: Exactly 10 requests accepted, exactly 10 rejected. Accepted requests are the first 10 chronologically. Rejection response is immediate (no queuing).

3. **Multi-key fairness under contention**
   - Duration: 8 goroutines each send 100 requests targeting a random selection of 50 keys (800 total requests, ~16 per key on average), with limit = 10 per key per window
   - Expected: Each key accepts at most 10 requests per window. No key is starved (all keys with requests receive at least 1 acceptance if they have requests before the limit is reached). Acceptance counts per key are deterministic (exactly min(requests_for_key, limit)).

4. **Window rotation correctness**
   - Duration: Send 10 requests for key A (filling the limit), wait for window rotation (110ms), then send 10 more for key A
   - Expected: All 20 requests accepted (10 per window). Counter resets cleanly at window boundary. No carry-over of counts from previous window.

5. **Rapid window transitions**
   - Duration: Over 10 consecutive windows (1 second total at 100ms windows), send exactly limit requests per key per window for 20 keys
   - Expected: 100% acceptance rate across all windows. No requests rejected due to stale window state. Counter reset happens atomically at each window boundary.

6. **High-throughput lookup performance**
   - Duration: Benchmark 1,000,000 rate limit checks across 100 keys using `testing.B`
   - Expected: Throughput exceeds 1,000,000 ops/sec on a single core. No lock contention visible in pprof. Per-check latency p99 < 1us.

### Test Implementation Notes

- Use short window durations (100ms) to enable multiple window rotations in a reasonable test time
- For fairness tests, record per-key accept/reject counts in a `map[string]int` protected by mutex
- Use `sync.WaitGroup` for concurrent goroutine coordination
- Verify window boundaries by checking that counters reset by sending requests in two consecutive windows
- Run with `-race` flag to detect races in concurrent counter updates
- Use `testing.B` for throughput benchmarks with `b.ResetTimer()` after setup

### Related Test Cases

- TC-PERF-002 (rate limiter gates delivery throughput)
- TC-PERF-006 (webhook handler uses rate limiting for request throttling)
