# TC-INT-011: Projector failure preserves previous dispatch table

**Priority:** P1
**Type:** Integration
**Package:** internal/hooks
**Related Tasks:** 07

## Objective

Validate that when the hook projector's `Apply` method fails (e.g., due to malformed data, transient error), the previous hook dispatch table remains intact and functional. The system must never leave the dispatch table in a broken or empty state due to a projector failure.

## Preconditions

- Real SQLite database via `t.TempDir()` with resource tables created
- Resource store initialized with hook projector
- An initial set of valid `hook.binding` resource records already reconciled into a working dispatch table
- Ability to inject a failure condition into the projector's Apply (e.g., corrupted record data, or a test projector wrapper that fails on demand)

## Test Steps

1. Persist 2 valid `hook.binding` records and trigger reconciliation.
   **Expected:** Projector Apply succeeds. Dispatch table contains 2 bindings.

2. Verify both hooks fire correctly by triggering their respective events.
   **Expected:** Both handlers invoked. Dispatch table is confirmed working.

3. Insert a malformed `hook.binding` record (e.g., invalid JSON in the data field) and trigger reconciliation.
   **Expected:** Projector Apply returns an error or logs a warning.

4. Verify the dispatch table still contains the original 2 bindings.
   **Expected:** Previous dispatch table is preserved. The malformed record did not corrupt or clear the table.

5. Trigger the events for both original hooks.
   **Expected:** Both handlers still fire correctly. Invocation counts increment as expected.

6. Fix the malformed record (update its data to valid JSON) and trigger reconciliation.
   **Expected:** Projector Apply succeeds. Dispatch table now contains 3 bindings (original 2 + the fixed one).

## Edge Cases

- Projector Apply panics (not just returns error) — recovered, previous table preserved
- All records are malformed in a single reconcile pass — entire Apply fails, previous table remains
- Projector Apply partially succeeds (processes some records before failing) — must be atomic: all or nothing
- Concurrent writes during a failed reconcile — next reconcile picks up all changes
- Repeated failures — dispatch table remains the last successfully reconciled version, never degraded
