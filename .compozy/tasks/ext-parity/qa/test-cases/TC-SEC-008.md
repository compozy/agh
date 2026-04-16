# TC-SEC-008: Snapshot Payload Limits Enforced

**Priority:** P1
**Type:** Security
**Package:** internal/resources
**Related Tasks:** 01

## Objective

Validate that `ApplySourceSnapshotRaw` rejects snapshot payloads whose total record count or cumulative size exceeds the configured per-call ceiling. Rejection must occur before any records in the snapshot are persisted.

## Preconditions

- The resource runtime is configured with snapshot limits (e.g., `MaxRecordsPerSnapshot=100`, `MaxSnapshotBytes=1MB`).
- Extension `ext-bulk` has an active session with a valid nonce.
- The resource store is empty or in a known state.

## Test Steps

1. As `ext-bulk`, submit a snapshot containing exactly `MaxRecordsPerSnapshot` (100) records, each within individual size limits.
   **Expected:** The snapshot is accepted and all 100 records are persisted.

2. As `ext-bulk`, submit a snapshot containing `MaxRecordsPerSnapshot + 1` (101) records.
   **Expected:** The snapshot is rejected with 413 Payload Too Large. The error message indicates the record count limit was exceeded.

3. Verify no records from step 2 were persisted.
   **Expected:** A `resources/list` for `ext-bulk`'s source shows only the 100 records from step 1. The rejected snapshot had zero effect.

4. As `ext-bulk`, submit a snapshot containing 10 records whose cumulative `spec_json` size exceeds `MaxSnapshotBytes` (1MB), even though each individual record is within its per-kind MaxBytes limit.
   **Expected:** The snapshot is rejected with 413 Payload Too Large. The error references the cumulative size limit, not individual record limits.

5. As `ext-bulk`, submit a snapshot with 0 records (empty snapshot).
   **Expected:** The snapshot is accepted. An empty snapshot is a valid operation that reconciles the source to having no records (deleting any existing ones).

## Edge Cases

- Snapshot where record count is within limits but the JSON envelope (array structure, metadata) pushes total payload size over the byte limit.
- Snapshot containing records with duplicate `(kind, id)` pairs -- verify deduplication does not reduce count below the limit check.
- Rapid sequential snapshots that individually are within limits but collectively represent a high write volume (rate limiting is a separate concern but interaction should be considered).
- Snapshot that is exactly at the byte limit boundary (off-by-one testing on cumulative size).
- Extension splits a large logical snapshot into multiple smaller snapshots to circumvent per-call limits -- verify that reconciliation semantics (each snapshot replaces all records for that source) prevent this from accumulating unbounded records.

## Threat Model

This test prevents **denial of service via snapshot flooding**. Without per-call ceilings, a malicious extension could submit a single snapshot containing millions of records or multi-gigabyte payloads, overwhelming the resource store, exhausting memory during processing, and potentially blocking other extensions' snapshots due to lock contention. The per-call limits bound the maximum resource consumption of any single snapshot operation, ensuring that the system remains responsive even when an extension misbehaves. The atomic rejection guarantee (no partial writes) prevents a scenario where a partially-applied oversized snapshot leaves the store in an inconsistent state.
