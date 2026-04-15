## TC-SEC-008: Rate Limiting Under Sustained Attack

**Priority:** P1
**Type:** Security
**Risk Level:** High
**Status:** Not Run
**Estimated Time:** 25 minutes
**Created:** 2026-04-15

---

### Objective

Verify that the fixed-window rate limiter correctly throttles excessive webhook requests per routing key, rejects excess requests with 429 status, does not starve legitimate traffic from other routing keys, and recovers properly when the attack subsides.

### Preconditions

- [ ] Bridge adapter runtime is running with rate limiting enabled
- [ ] Rate limit threshold is known (e.g., 100 requests per 60-second window per routing key)
- [ ] At least two distinct routing keys are available for testing (e.g., two different instance IDs or sender IDs)
- [ ] HTTP client capable of high-concurrency request sending is available (e.g., `hey`, `wrk`, or custom Go test harness)
- [ ] Timing measurement capability for response latency analysis

### Test Steps

1. **Requests within rate limit — all accepted**
   - Input: Send requests at 50% of the rate limit threshold from routing key `key-A` within one window (e.g., 50 requests if limit is 100).
   - **Expected:** All requests return 200 OK (assuming valid signatures). No rate limiting headers indicate throttling.

2. **Requests at exact rate limit — all accepted**
   - Input: Send exactly the rate limit threshold number of requests from `key-A` within one window.
   - **Expected:** All requests accepted. The last request is at the boundary but still within the limit.

3. **Requests exceeding rate limit — excess rejected**
   - Input: Send 150% of the rate limit threshold from `key-A` within one window (e.g., 150 requests if limit is 100).
   - **Expected:** First 100 requests return 200 OK. Requests 101-150 return 429 Too Many Requests. Response includes `Retry-After` header or similar indication of when the client can retry.

4. **Sustained attack — 1000 requests from single key**
   - Input: Send 1000 requests rapidly from `key-A` within one window.
   - **Expected:** Only the first N requests (up to the limit) are accepted. Remaining 1000-N requests return 429. Server remains responsive throughout. No CPU spike or memory exhaustion from rate limiter bookkeeping.

5. **Cross-key isolation — attacked key does not starve others**
   - Input: While sending 1000 requests from `key-A` (exceeding its limit), simultaneously send 10 requests from `key-B`.
   - **Expected:** `key-A` requests are throttled (429 after limit). All 10 `key-B` requests return 200 OK. Rate limiting is per-key, not global.

6. **Window reset — requests accepted after window expires**
   - Input: (a) Send requests exceeding the limit for `key-A` in window 1. (b) Wait for the window to expire. (c) Send a single request from `key-A`.
   - **Expected:** (a) Excess requests rejected with 429. (b) Window expires. (c) Request accepted with 200 OK. Counter is reset.

7. **Rate limit response body**
   - Input: Trigger a 429 response.
   - **Expected:** Response body contains a structured error message (e.g., `{"error":"rate limit exceeded"}`). Response does not leak internal rate limit state, routing key details, or other sensitive information.

8. **Rate limiter does not block health checks**
   - Input: While `key-A` is rate-limited, send a request to the health/readiness endpoint.
   - **Expected:** Health endpoint returns 200 OK. Rate limiting applies only to webhook endpoints, not operational endpoints.

9. **Rate limit ordering in the security pipeline**
   - Input: Send a request that would be rate-limited (429) but also has an invalid HTTP method (GET instead of POST).
   - **Expected:** 405 Method Not Allowed (not 429). Method validation occurs before rate limiting in the pipeline. This confirms the ordering: method -> content-type -> body size -> rate limit -> signature.

10. **Rate limiter memory bounds under many unique keys**
    - Input: Send one request each from 10,000 unique routing keys within one window.
    - **Expected:** All requests accepted (each key is within its individual limit). Rate limiter memory usage grows linearly but stays bounded. Old keys are eventually cleaned up (verify after several windows expire).

11. **Concurrent rate limit counter accuracy**
    - Input: Send exactly the rate limit threshold number of requests from `key-A` using 50 concurrent goroutines/threads, all within one window.
    - **Expected:** Exactly the threshold number of requests are accepted. The counter is accurate under concurrent access (no race condition allowing more requests than the limit).

12. **Fixed-window boundary behavior**
    - Input: Send 80% of the limit at the end of window 1, then 80% of the limit at the start of window 2 (within a few milliseconds of the window boundary).
    - **Expected:** Both batches are accepted (each is within its respective window). This is the expected behavior of a fixed-window algorithm. Document if sliding-window behavior is observed instead.

### Attack Vectors

- [ ] Volumetric denial-of-service via rapid request flooding from a single source
- [ ] Distributed attack using many routing keys to exhaust global resources
- [ ] Rate limiter memory exhaustion via millions of unique routing keys
- [ ] Race condition in counter increment allowing more requests than the limit
- [ ] Window boundary exploitation to get 2x the rate limit in a short burst
- [ ] Rate limit bypass by manipulating routing key extraction (e.g., header spoofing)
- [ ] Starvation of legitimate users when rate limiter state is shared globally

### Related Test Cases

- TC-SEC-003 (Method validation — occurs before rate limiting)
- TC-SEC-004 (Body size limits — occurs before rate limiting)
- TC-SEC-009 (In-flight concurrency — complementary protection against concurrent load)
- TC-SEC-001 (Signature verification — occurs after rate limiting)
