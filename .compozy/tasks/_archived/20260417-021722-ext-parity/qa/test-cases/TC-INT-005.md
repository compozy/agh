# TC-INT-005: Close() drains in-flight work within deadline

**Priority:** P0
**Type:** Integration
**Package:** internal/resources
**Related Tasks:** 03

## Objective

Validate that calling `Close(ctx)` on the resource runtime drains in-flight projector work within the context deadline. A slow projector must be cancelled, no goroutines should leak after Close returns, and the system must shut down cleanly.

## Preconditions

- Real SQLite database via `t.TempDir()` with resource tables created
- Resource store initialized with a reconcile driver
- A slow projector registered that blocks for 5 seconds in its Apply method (simulating heavy work)
- The slow projector respects context cancellation

## Test Steps

1. Register a slow projector for `kind=tool` that sleeps for 5s in Apply but checks `ctx.Done()`.
   **Expected:** Projector registered and ready.

2. Write a resource record to trigger the slow projector.
   **Expected:** Write accepted. Projector Apply starts executing (blocks on the 5s sleep).

3. After a short delay (100ms, enough for Apply to start), call `Close(ctx)` with a 2s deadline context.
   **Expected:** Close begins draining. The slow projector's context is cancelled.

4. Measure the wall-clock time for Close to return.
   **Expected:** Close returns well before 5s (the projector's full sleep). It should return within the 2s deadline.

5. Verify the slow projector observed context cancellation (e.g., via a channel or flag set in the projector).
   **Expected:** The projector's cancellation flag is set, confirming it was interrupted.

6. Verify no goroutine leaks by checking runtime goroutine count or using `goleak` if available.
   **Expected:** No leaked goroutines from the resource runtime.

7. Attempt to write another record after Close.
   **Expected:** Write returns an error (store is closed/shut down).

## Edge Cases

- Close with an already-expired context — returns immediately with context error, no new work started
- Close with no in-flight work — returns instantly
- Multiple concurrent Close calls — idempotent, no panic or double-close errors
- Projector that ignores context cancellation — Close still returns at deadline, goroutine is orphaned but runtime itself is shut down
- Close during a coalescing window — pending coalesced batch is either executed or dropped cleanly
