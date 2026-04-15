## TC-PERF-006: Webhook Concurrent Request Handling

**Priority:** P2
**Type:** Performance
**Status:** Not Run
**Estimated Time:** 20 minutes
**Created:** 2026-04-15

---

### Objective

Verify the Webhook HTTP server handles concurrent requests correctly, the InFlightLimiter enforces its concurrency cap, excess requests receive 503 responses, and no goroutine leaks occur after all requests complete.

### Preconditions

- [ ] Webhook HTTP server is running on a test port (localhost)
- [ ] InFlightLimiter is configured with a known concurrency cap (e.g., 20 concurrent requests)
- [ ] A provider endpoint is registered that introduces artificial latency (e.g., 100ms per request) to ensure in-flight overlap
- [ ] Goroutine counting baseline is recorded before test starts
- [ ] HTTP client is configured with no connection pooling limits that would artificially serialize requests
- [ ] Test timeout is set to 60 seconds to accommodate all scenarios

### Performance Criteria

| Metric                                         | Target                          | Acceptable         | Actual | Status |
| ---------------------------------------------- | ------------------------------- | ------------------ | ------ | ------ |
| Successful requests (within concurrency cap)   | >= concurrency cap              | >= concurrency cap |        |        |
| Rejected requests (503 status)                 | total - cap (when cap exceeded) | total - cap +/- 5  |        |        |
| Request latency p50 (accepted requests)        | < 150ms                         | < 250ms            |        |        |
| Request latency p95 (accepted requests)        | < 200ms                         | < 500ms            |        |        |
| Request latency for 503 rejections             | < 5ms                           | < 20ms             |        |        |
| Goroutine count after all requests complete    | baseline +/- 2                  | baseline +/- 5     |        |        |
| Goroutine count 5s after all requests complete | baseline +/- 1                  | baseline +/- 2     |        |        |
| Connection leaks (open file descriptors)       | 0                               | 0                  |        |        |

### Load Scenarios

1. **Within concurrency cap**
   - Duration: Send 15 concurrent requests to a provider endpoint with concurrency cap = 20
   - Expected: All 15 requests succeed with 200 status. No 503 rejections. All requests complete within expected handler latency + overhead. Goroutine count returns to baseline after completion.

2. **Exceeding concurrency cap**
   - Duration: Send 200 concurrent requests to a single provider endpoint with concurrency cap = 20, handler latency = 100ms
   - Expected: Exactly 20 requests are processed concurrently (in-flight at any point). Remaining requests receive 503 immediately (< 5ms response time). Successful request count is >= 20 (depends on how quickly the first batch completes and allows new requests). All 200 responses are received (no hangs).

3. **Burst followed by drain**
   - Duration: Send 100 requests in a single burst, then wait for all to complete, then send 10 more
   - Expected: First burst: concurrency cap enforced, mix of 200 and 503 responses. After drain: all 10 follow-up requests succeed with 200 (semaphore fully released). Goroutine count returns to baseline between bursts.

4. **Sustained load at cap boundary**
   - Duration: For 10 seconds, maintain exactly 20 concurrent requests (replace each completed request with a new one)
   - Expected: Near-100% success rate (brief windows of 503 acceptable during replacement). Throughput remains stable. No goroutine growth over time. Memory footprint stable.

5. **Request body handling under load**
   - Duration: Send 50 concurrent requests each with a 10KB JSON body, concurrency cap = 20
   - Expected: All accepted requests correctly parse the request body. No truncated or corrupted bodies. Rejected requests (503) do not consume the request body unnecessarily. No memory spike from buffering rejected request bodies.

6. **Goroutine leak stress test**
   - Duration: Run 5 rounds of 200 concurrent requests each, with 1 second pause between rounds. Record goroutine count after each round.
   - Expected: Goroutine count after each round returns to within 2 of baseline. No monotonic growth across rounds. After final round + 5 second wait, goroutine count equals baseline. No leaked HTTP handler goroutines, no leaked limiter goroutines.

### Test Implementation Notes

- Use `net/http/httptest.NewServer` for the webhook server to get an ephemeral port
- Use `sync.WaitGroup` and a channel-based semaphore to coordinate concurrent request sending
- Send requests using `http.Client` with `Transport` configured for high `MaxIdleConnsPerHost` to avoid client-side bottlenecks
- Record `runtime.NumGoroutine()` at multiple checkpoints: before test, after each scenario, and after a cooldown period
- For the sustained load test, use a worker pool pattern: N goroutines each send requests in a loop, replacing completed ones
- Verify 503 responses have appropriate response body/headers (not just status code)
- Use `net.Dial` or `/proc/self/fd` inspection (Linux) or `lsof` (macOS) to check for connection leaks if feasible
- Run with `-race` flag to detect races in the in-flight limiter's semaphore operations

### Related Test Cases

- TC-PERF-002 (webhook requests feed into delivery broker)
- TC-PERF-004 (rate limiter may gate webhook request processing)
- TC-PERF-005 (webhook handler may access instance cache)
