# TC-FUNC-010: Degraded circuit after repeated failures

**Priority:** P1
**Type:** Functional
**Package:** internal/resources
**Related Tasks:** 03

## Objective

Validate that the reconciliation engine implements a degraded circuit breaker pattern. After a projector fails a configurable number of consecutive times (e.g., 3), the circuit opens and suppresses further automatic reruns for that kind. When a new committed write occurs for that kind, the circuit resets its backoff and allows a fresh reconciliation attempt. This prevents infinite failure loops while ensuring recovery is possible.

## Preconditions

- A fresh resource store is initialized with the reconciliation engine.
- A projector for the `tool` kind is registered. The projector is configurable to either succeed or return an error on demand (e.g., via a shared atomic flag).
- The circuit breaker threshold is configured to 3 consecutive failures.
- The backoff parameters are configured with short durations for test speed (e.g., initial backoff 10ms, max backoff 100ms).

## Test Steps

1. Configure the projector to always fail. Trigger reconciliation for the `tool` kind.
   **Expected:** The projector is invoked and fails. The failure is logged. The engine schedules a retry with backoff.

2. Allow the engine to retry automatically until 3 consecutive failures have occurred.
   **Expected:** After the 3rd failure, the circuit transitions to the `degraded` (open) state. Subsequent automatic retries are suppressed. An observable metric or log entry indicates the circuit is open for the `tool` kind.

3. Trigger reconciliation for the `tool` kind externally (e.g., via an explicit API call or additional record commit).
   **Expected:** While the circuit is open, the trigger is either queued with extended backoff or suppressed entirely, depending on the circuit policy. The projector is NOT immediately re-invoked.

4. Commit a new `tool` record to the store via `PutRaw`.
   **Expected:** The committed write resets the circuit breaker for the `tool` kind. The backoff timer is cleared. The circuit transitions back to `closed` (or `half-open`).

5. Configure the projector to succeed. Wait for the next reconciliation attempt.
   **Expected:** The projector is invoked and succeeds. The circuit remains closed. The consecutive failure counter resets to 0.

6. Configure the projector to fail again. Verify it takes another 3 consecutive failures to open the circuit.
   **Expected:** The circuit opens again only after 3 new consecutive failures, confirming the counter was fully reset by the successful run in step 5.

## Edge Cases

- A projector that alternates success/failure (fail, succeed, fail, succeed) never opens the circuit because consecutive failures never reach the threshold.
- Circuit opens for `tool` kind but `skill` kind remains healthy: the two circuits are independent.
- All circuits are open: the reconciliation engine remains idle but is still responsive to new committed writes.
- The backoff duration between retries increases exponentially up to the configured max, and a successful retry resets the backoff to the initial value.
- Shutdown while the circuit is open: the engine shuts down cleanly without attempting pending retries.
- The circuit state is observable (e.g., via a health endpoint or structured log) so operators can detect degraded kinds.
