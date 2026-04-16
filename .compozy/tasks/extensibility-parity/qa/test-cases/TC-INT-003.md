# TC-INT-003: Source reset removes records and source state atomically

**Priority:** P1
**Type:** Integration
**Package:** internal/resources
**Related Tasks:** 01

## Objective

Validate that an operator-initiated source reset deletes both the source-owned resource records and the corresponding `resource_source_state` row within a single atomic transaction. Partial cleanup (records deleted but state retained, or vice versa) must never occur.

## Preconditions

- Real SQLite database via `t.TempDir()` with resource tables created
- Resource store initialized
- Source `ext-alpha` has published records and has a `resource_source_state` entry with a valid nonce
- At least one other source (`ext-beta`) also has records to verify isolation

## Test Steps

1. Publish several resource records from `source=ext-alpha` (e.g., 3 tool records and 2 hook.binding records).
   **Expected:** All 5 records persisted. `resource_source_state` row exists for `ext-alpha`.

2. Publish several resource records from `source=ext-beta` (e.g., 2 tool records).
   **Expected:** All 2 records persisted. `resource_source_state` row exists for `ext-beta`.

3. Execute operator source reset for `ext-alpha`.
   **Expected:** Operation completes without error.

4. Query `resource_records` for `source=ext-alpha`.
   **Expected:** Zero rows returned. All 5 records are gone.

5. Query `resource_source_state` for `source=ext-alpha`.
   **Expected:** Zero rows returned. The nonce/state row is gone.

6. Query `resource_records` for `source=ext-beta`.
   **Expected:** All 2 records still present and unchanged.

7. Query `resource_source_state` for `source=ext-beta`.
   **Expected:** Row still present with original nonce.

## Edge Cases

- Reset a source that has no records — no error, no-op
- Reset a source that has records but no source_state entry — records still removed, no error
- Concurrent reset of the same source — both complete without deadlock or partial state
- Reset during an in-flight snapshot from the same source — snapshot should fail with stale nonce after reset completes
- Verify the operation uses a single SQLite transaction (check via WAL or transaction count if feasible)
