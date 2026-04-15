## TC-SEC-009: In-Flight Concurrency Limiting

**Priority:** P1
**Type:** Security
**Risk Level:** High
**Status:** Not Run
**Estimated Time:** 25 minutes
**Created:** 2026-04-15

---

### Objective
Verify that the in-flight concurrency limiter correctly caps the number of simultaneously processing webhook requests, rejects excess concurrent requests with 503, properly decrements the counter upon completion (success or failure), and recovers gracefully when load subsides.

### Preconditions
- [ ] Bridge adapter runtime is running with in-flight concurrency limiting enabled
- [ ] In-flight concurrency limit is known (e.g., 50 concurrent requests)
- [ ] HTTP client capable of managing concurrent connections with precise control is available
- [ ] A slow webhook handler or artificial processing delay can be introduced for testing (e.g., a provider that takes 2 seconds to process each webhook)
- [ ] Monitoring for in-flight counter state is available (metrics endpoint or logs)

### Test Steps

1. **Concurrent requests within limit — all accepted**
   - Input: Send 25 concurrent requests (50% of a 50-request limit) that each take 2 seconds to process. All requests are in-flight simultaneously.
   - **Expected:** All 25 requests are accepted and processed. No 503 responses. All return 200 OK after processing.

2. **Concurrent requests at exact limit — all accepted**
   - Input: Send exactly 50 concurrent requests, all held in-flight simultaneously (each takes 2 seconds).
   - **Expected:** All 50 requests accepted. The in-flight counter reaches exactly the limit. All return 200 OK.

3. **Concurrent requests exceeding limit — excess rejected**
   - Input: Send 70 concurrent requests, all arriving within milliseconds. Each takes 2 seconds to process.
   - **Expected:** First 50 requests are accepted and begin processing. Requests 51-70 receive 503 Service Unavailable immediately (not after a timeout). The 503 response is returned quickly, not after waiting for a slot.

4. **503 response format and headers**
   - Input: Trigger a 503 response by exceeding the in-flight limit.
   - **Expected:** Response body contains a structured error (e.g., `{"error":"service temporarily unavailable"}`). Response may include `Retry-After` header. Response does not leak the in-flight limit value or current counter.

5. **Counter decrement on successful completion**
   - Input: (a) Fill the in-flight limit to capacity (50 concurrent requests, each taking 2 seconds). (b) Wait for all 50 to complete. (c) Immediately send 1 new request.
   - **Expected:** (a) All 50 accepted. (b) All complete with 200 OK. (c) New request accepted with 200 OK. Counter has decremented back to 0 after completions.

6. **Counter decrement on error completion**
   - Input: (a) Fill the in-flight limit with 50 requests, half of which will fail at signature verification (returning 401). (b) Wait for all to complete. (c) Send 1 new request.
   - **Expected:** (a) All 50 accepted into processing. 25 return 200, 25 return 401. (b) All complete. (c) New request accepted. Counter correctly decremented for both successful and failed requests (no counter leak on error paths).

7. **Counter decrement on panic/crash recovery**
   - Input: (a) Fill the in-flight limit. (b) Simulate a handler panic during processing of one request (if testable). (c) After recovery, send a new request.
   - **Expected:** Panic is recovered. In-flight counter is decremented for the panicked request (via deferred decrement). New request is accepted. No permanent counter leak.

8. **Rapid burst followed by recovery**
   - Input: (a) Send 200 requests in a rapid burst (4x the limit). (b) Wait for all processing to complete. (c) Send 10 requests.
   - **Expected:** (a) ~50 accepted, ~150 rejected with 503. (b) Processing completes. (c) All 10 new requests accepted. System fully recovers.

9. **Interaction with rate limiting**
   - Input: Send 200 concurrent requests from the same routing key, exceeding both the rate limit and the in-flight limit.
   - **Expected:** Requests are first checked against the rate limit (429 for excess), then against the in-flight limit (503 for excess). The order of rejection depends on pipeline ordering. Both limits are enforced independently.

10. **Long-running request does not permanently consume a slot**
    - Input: Send a request that takes 30 seconds to process (slow provider). While it's processing, fill the remaining in-flight slots. Wait for the slow request to timeout or complete.
    - **Expected:** The slow request eventually completes or times out. Its slot is released. No permanent consumption of in-flight capacity by hung requests.

11. **In-flight limit per-instance vs global**
    - Input: If in-flight limiting is per-instance: fill Instance A's in-flight limit, then send requests to Instance B.
    - **Expected:** Instance B's requests are accepted (Instance A's limit does not affect Instance B). If global: document the behavior and verify total in-flight across all instances respects the global limit.

12. **Concurrent counter thread safety**
    - Input: Send 1000 requests in a rapid burst with high concurrency (100+ goroutines). Track accept/reject counts.
    - **Expected:** Total accepted requests never exceed the in-flight limit at any point in time. `accepted + rejected = 1000`. No race conditions in counter increment/decrement (verified by `-race` flag in Go tests).

### Attack Vectors
- [ ] Connection exhaustion by sending many slow requests that hold in-flight slots
- [ ] Counter leak via error paths that skip decrement (permanently reducing capacity)
- [ ] Panic in handler causing counter to not decrement
- [ ] Slowloris-style attacks holding connections open to exhaust in-flight capacity
- [ ] Thundering herd after recovery — all retrying clients hitting at once
- [ ] Race conditions in concurrent counter access allowing more in-flight than the limit
- [ ] Interaction bypass — exceeding in-flight limit when rate limiter is also active

### Related Test Cases
- TC-SEC-004 (Body size limits — large bodies increase processing time, interacting with in-flight limits)
- TC-SEC-008 (Rate limiting — complementary protection, different axis of limiting)
- TC-SEC-003 (Method validation — occurs before in-flight tracking)
