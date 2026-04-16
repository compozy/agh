# TC-FUNC-009: Single-flight reconcile per kind

**Priority:** P0
**Type:** Functional
**Package:** internal/resources
**Related Tasks:** 03

## Objective

Validate that the reconciliation engine enforces single-flight execution per resource kind. When the same kind is triggered for reconciliation multiple times rapidly, at most one reconciliation is in-flight at any time, with at most one queued rerun. This prevents thundering-herd reconciliation storms and ensures efficient use of resources while still guaranteeing eventual consistency.

## Preconditions

- A fresh resource store is initialized with schema applied, including the reconciliation engine.
- A projector for the `tool` kind is registered. The projector includes an artificial delay (e.g., 100ms) to simulate real work and allow overlap detection.
- A concurrency-safe counter tracks how many projector invocations are executing simultaneously.
- The reconciliation engine is started and ready to accept triggers.

## Test Steps

1. Trigger reconciliation for the `tool` kind 10 times in rapid succession (within <10ms total), using a loop or fan-out of goroutines.
   **Expected:** All 10 trigger calls return without error (triggers are non-blocking fire-and-forget).

2. Wait for all reconciliation activity to settle (e.g., wait for the projector's artificial delay plus a buffer).
   **Expected:** The maximum concurrent projector invocations observed for the `tool` kind is exactly 1. At no point were two projector callbacks running simultaneously for the same kind.

3. Count the total number of projector invocations for the `tool` kind.
   **Expected:** The total is at most 2: one for the initial trigger, and one queued rerun that coalesced the remaining 9 triggers. It may be exactly 1 if the first pass completed before any rerun was needed.

4. Trigger reconciliation for the `tool` kind once more after quiescence.
   **Expected:** Exactly one more projector invocation occurs, confirming the engine is still responsive after the coalescing burst.

5. Trigger reconciliation for two different kinds (`tool` and `skill`) simultaneously.
   **Expected:** Both kinds reconcile concurrently (they are independent). The single-flight constraint is per-kind, not global.

## Edge Cases

- A projector that panics during execution: the single-flight slot is released, the panic is recovered, and subsequent triggers can still execute.
- A projector that takes an extremely long time (e.g., 30s): the queued rerun waits until completion, and no additional reruns are queued beyond the one.
- Triggering a kind that has no registered projector: the trigger is a no-op or returns a clear error, and does not consume a flight slot.
- Shutdown signal arrives while a reconciliation is in-flight: the in-flight pass is allowed to complete (or cancelled via context), and the queued rerun is discarded.
- The rerun carries the latest store state, not a stale snapshot from when it was queued.
