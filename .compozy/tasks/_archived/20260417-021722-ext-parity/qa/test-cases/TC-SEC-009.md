# TC-SEC-009: Rate Limiting on Direct Operator Writes

**Priority:** P1
**Type:** Security
**Package:** internal/api/udsapi
**Related Tasks:** 06

## Objective

Validate that the UDS API enforces rate limiting on direct operator write operations (PUT/DELETE on resources). Requests exceeding the configured rate limit must receive 429 Too Many Requests responses, and the rate limiter must not block legitimate traffic once the window resets.

## Preconditions

- The daemon is running with UDS API enabled.
- Rate limiting is configured for operator write operations (e.g., `MaxWritesPerSecond=10` or equivalent token-bucket/sliding-window configuration).
- An operator client is connected via the Unix domain socket.
- The resource store contains at least one record for deletion tests.

## Test Steps

1. Send a single PUT request to create a resource record via UDS.
   **Expected:** The request succeeds with 200/201. A single request within the rate limit passes normally.

2. Send a burst of PUT requests at a rate exceeding the configured limit (e.g., 20 requests in 1 second when the limit is 10/second).
   **Expected:** The first 10 requests succeed. Requests 11-20 are rejected with 429 Too Many Requests. The 429 response includes a `Retry-After` header or equivalent indication of when the client can retry.

3. Wait for the rate limit window to reset, then send a single PUT request.
   **Expected:** The request succeeds. The rate limiter correctly resets and does not permanently block the client.

4. Send a burst of DELETE requests at a rate exceeding the configured limit.
   **Expected:** Same behavior as step 2 -- DELETE operations are subject to the same rate limits as PUT operations.

5. Interleave PUT and DELETE requests to verify that both operation types share the same rate limit budget (or have separate budgets, depending on configuration).
   **Expected:** The combined rate of write operations does not exceed the configured limit. The behavior matches the documented rate limit policy.

## Edge Cases

- Multiple operator clients connected simultaneously -- verify rate limiting is per-client, per-source, or global as configured.
- Read operations (GET, LIST) interspersed with writes -- verify reads are not rate-limited by the write limiter.
- Client sends requests with `Connection: keep-alive` vs new connections per request -- rate limiting should apply regardless of connection strategy.
- Rate limit configuration set to 0 (disabled) -- verify all requests pass without 429.
- Rate limit window boundary: send a request at the exact moment the window resets to verify no off-by-one in window calculation.

## Threat Model

This test prevents **resource exhaustion via operator write flooding**. The UDS API is the direct operator interface, accessible to any process that can reach the Unix domain socket. Without rate limiting, a compromised CLI tool, a runaway script, or a malicious local process could flood the resource store with write operations, causing excessive disk I/O, lock contention that blocks extension snapshots, and potential SQLite write-ahead log growth that degrades overall daemon performance. Rate limiting ensures that even with UDS access, no single client can monopolize write throughput.
