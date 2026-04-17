# TC-SEC-005: Stale Session Nonce Rejected Before Source Lock

**Priority:** P0
**Type:** Security
**Package:** internal/resources
**Related Tasks:** 01, 05

## Objective

Validate that a snapshot submitted with a non-active (stale, expired, or fabricated) session nonce is rejected before the per-source serialization lock is acquired. This prevents both unauthorized writes and timing side-channel attacks that could reveal lock contention patterns.

## Preconditions

- Extension `ext-target` is registered with an active session and a valid nonce `nonce-A`.
- The session is cycled so `nonce-A` becomes stale and a new `nonce-B` is active.
- A known resource record exists for `ext-target`.
- Instrumentation or logging is available to observe lock acquisition order.

## Test Steps

1. Submit a snapshot using the stale `nonce-A` for source `ext-target`.
   **Expected:** The snapshot is rejected immediately with an authentication/authorization error (e.g., 401 or 403). The error occurs before any per-source lock is acquired.

2. Measure the response time of the rejected request from step 1.
   **Expected:** The response time is consistent regardless of whether the per-source lock is currently held by another operation. There is no observable timing difference that would indicate lock contention.

3. Submit a snapshot using a completely fabricated nonce (`nonce-fake-12345`) for source `ext-target`.
   **Expected:** Rejected with the same error code and comparable response time as step 1. No distinction between "stale" and "never-existed" nonces in the error response.

4. Submit a snapshot using the valid `nonce-B` for source `ext-target`.
   **Expected:** The snapshot succeeds, confirming the active nonce path works correctly.

5. Concurrently submit two requests: one valid snapshot (with `nonce-B`) and one with stale `nonce-A`, both targeting the same source.
   **Expected:** The stale-nonce request is rejected without blocking or delaying the valid request. The valid request completes successfully.

## Edge Cases

- Nonce from a different extension's session is used (cross-session nonce replay).
- Empty nonce string submitted in the snapshot payload.
- Nonce field omitted entirely from the snapshot request.
- Nonce that was valid in a previous daemon lifecycle (daemon restart scenario).
- Rapid alternation between stale and valid nonces to probe for timing differences.

## Threat Model

This test prevents two attack vectors: (1) **Stale credential replay** -- a captured or leaked nonce from a previous session cycle cannot be used to submit unauthorized snapshots, even if the attacker knows the source identifier. (2) **Timing side-channel on lock contention** -- if nonce validation occurred after lock acquisition, an attacker could submit requests with invalid nonces and measure response times to determine whether the legitimate extension is currently performing a snapshot (lock held = slower rejection). This information leakage could be used to time attacks or infer extension activity patterns.
