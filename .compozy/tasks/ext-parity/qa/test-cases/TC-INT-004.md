# TC-INT-004: Write storm coalesces within bounded window

**Priority:** P1
**Type:** Integration
**Package:** internal/resources
**Related Tasks:** 03

## Objective

Validate that the reconcile driver coalesces rapid writes within a bounded time window. When 50 writes for a single kind arrive within 10ms, the reconcile driver should execute at most 3 passes total, demonstrating effective coalescing rather than per-write reconciliation.

## Preconditions

- Real SQLite database via `t.TempDir()` with resource tables created
- Resource store initialized with a reconcile driver wired for the target kind
- A test projector (or mock projector implementing the real interface) that counts Apply invocations via an atomic counter
- Coalescing window configured (use default or explicitly set to a known value)

## Test Steps

1. Register a counting projector for `kind=tool` that increments an atomic counter on each `Apply` call.
   **Expected:** Projector registered. Counter starts at 0.

2. Fire 50 sequential writes for `kind=tool` with distinct IDs, all within a tight loop (~10ms total).
   **Expected:** All 50 writes accepted without error.

3. Wait for the coalescing window to close plus a small buffer (e.g., 100ms after last write).
   **Expected:** All writes have been processed.

4. Read the projector's invocation counter.
   **Expected:** Counter is <= 3. The reconcile driver batched the 50 writes into at most 3 reconcile passes.

5. Verify all 50 records exist in the resource store.
   **Expected:** All 50 records present with correct data.

## Edge Cases

- Writes for different kinds interleaved — each kind's projector coalesces independently
- Write arrives exactly at the coalescing window boundary — included in current batch or next batch, never dropped
- Projector Apply takes longer than the coalescing window — next batch waits for completion, no concurrent Apply for the same kind
- Zero writes after initial burst — no spurious reconcile passes
- Verify the coalescing bound holds under `-race` (no data races in the counter or driver)
