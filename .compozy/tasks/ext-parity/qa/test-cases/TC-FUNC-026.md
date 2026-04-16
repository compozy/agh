# TC-FUNC-026: Bridge projector Apply degrades on side-effect failure

**Priority:** P1
**Type:** Functional
**Package:** internal/bridges
**Related Tasks:** 11

## Objective

Validate that when the bridge projector's Apply encounters a failure while converging a live bridge connection (e.g., network unreachable, authentication rejected, timeout), the runtime marks the affected bridge instance as degraded rather than corrupting the entire bridge registry. Healthy bridges must remain fully operational, and the degraded bridge must be visible in the registry with a clear status for observability and retry.

## Preconditions

- Bridge projector has completed a Build producing a delta plan that includes: adding "ext-slack" (endpoint is deliberately unreachable) and updating "ext-github" (endpoint is valid and reachable).
- The live bridge registry currently has "ext-github" connected with the old endpoint.
- Network conditions are configured so that "ext-slack" connection will fail (e.g., mock transport returns connection refused).

## Test Steps

1. Call Apply with the Build result containing both "ext-slack" (will fail) and "ext-github" (will succeed).
   **Expected:** Apply completes without panicking. It does not return a blanket fatal error.

2. Query the live bridge registry for "ext-github".
   **Expected:** "ext-github" is connected with the new endpoint URL. Status is healthy/connected. The successful update was applied despite the other bridge failing.

3. Query the live bridge registry for "ext-slack".
   **Expected:** "ext-slack" is present in the registry with a degraded (or error/pending) status. The entry includes an error message or reason (e.g., "connection refused"). It is NOT absent — it is tracked for observability.

4. Verify the overall registry integrity.
   **Expected:** No stale entries, no duplicate entries, no nil pointers. The registry is in a consistent state. Pre-existing bridges that were not part of the delta plan are unaffected.

5. Attempt to use "ext-github" for a bridge operation (e.g., send a message through the bridge).
   **Expected:** Operation succeeds. The healthy bridge is fully functional.

6. Attempt to use "ext-slack" for a bridge operation.
   **Expected:** Operation fails with a clear error indicating the bridge is degraded. The failure does not cascade to other bridges.

7. Fix the network condition for "ext-slack" (make the endpoint reachable). Run a new Build + Apply cycle.
   **Expected:** Build detects that "ext-slack" needs convergence (still in degraded state). Apply successfully connects "ext-slack". Status transitions from degraded to connected.

8. Verify both bridges are now healthy.
   **Expected:** Registry shows "ext-github" connected and "ext-slack" connected. No degraded entries.

## Edge Cases

- All bridges in the delta plan fail during Apply: registry retains the previous healthy state for unchanged bridges. All new/updated bridges are marked degraded. No healthy bridge is lost.
- Apply timeout for a slow bridge (hangs instead of failing fast): Apply enforces a per-bridge connection timeout. Timed-out bridges are marked degraded, not left in a connecting state indefinitely.
- Degraded bridge that intermittently reconnects on its own (without Build+Apply): verify whether auto-recovery is supported or if explicit Build+Apply is required to re-converge.
- Multiple consecutive Apply calls with the same degraded bridge: no resource leak (connections, goroutines, file descriptors) from repeated failed connection attempts.
- Bridge that fails during Apply after partial setup (e.g., TCP connected but auth failed): partial state is cleaned up. No half-open connections remain.
