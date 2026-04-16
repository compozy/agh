# TC-FUNC-004: Source snapshot serialization

**Priority:** P1
**Type:** Functional
**Package:** internal/resources
**Related Tasks:** 01

## Objective

Validate that `ApplySourceSnapshotRaw` correctly serializes concurrent calls for the same source. When two snapshot operations arrive simultaneously for the same source, they must execute sequentially (not interleave), and a snapshot with a stale `source_version` must be rejected. This ensures atomic bulk reconciliation without partial application.

## Preconditions

- A fresh resource store is initialized with schema applied.
- A valid `MutationActor` with `SourceKind="extension"` and `SourceID="ext-A"` is configured.
- Multiple resource kinds are registered (e.g., `tool`, `skill`).
- Pre-existing records for source `ext-A` exist at a known `source_version`.

## Test Steps

1. Seed the store with 3 records from source `ext-A` at `source_version=1`: `tool/t1`, `tool/t2`, `skill/s1`.
   **Expected:** All 3 records are created successfully. The source version is tracked at `1`.

2. Launch two goroutines simultaneously, each calling `ApplySourceSnapshotRaw` for source `ext-A`:
   - Goroutine A: `source_version=2`, snapshot contains `tool/t1` (updated), `tool/t3` (new), `skill/s1` (unchanged). Removes `tool/t2`.
   - Goroutine B: `source_version=3`, snapshot contains `tool/t1` (updated differently), `tool/t4` (new).
   **Expected:** Both calls complete without panics or data corruption. The calls are serialized: one completes fully before the other starts. The final source version is `3` (or `2` if B was rejected due to requiring `source_version=2` as prerequisite).

3. Read all records for source `ext-A`.
   **Expected:** The store state is consistent with the serialized application order. No partial snapshot state is visible (e.g., no state where `t3` exists from goroutine A but `t4` also exists from goroutine B's partial application).

4. Attempt `ApplySourceSnapshotRaw` with `source_version=1` (now stale).
   **Expected:** The call is rejected with an error indicating the source version is stale. No records are modified.

5. Verify record-level versions after the snapshots.
   **Expected:** Each record that was created or updated by a snapshot has its own record-level version incremented appropriately, independent of the source version.

## Edge Cases

- Snapshot with an empty record set for a source that has existing records: all existing records from that source are deleted (full retraction).
- Snapshot containing a record kind that the source is not authorized to publish: the entire snapshot is rejected atomically.
- Snapshot with `source_version=0` on a source that already has records: rejected as stale, not treated as a "create" operation.
- Very large snapshot (hundreds of records) completes within a reasonable timeout and does not hold the serialization lock long enough to starve other sources.
- Two different sources (`ext-A` and `ext-B`) applying snapshots concurrently do not block each other, since serialization is per-source.
