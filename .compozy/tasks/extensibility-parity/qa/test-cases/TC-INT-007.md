# TC-INT-007: Extension snapshot publishes and reads back own records

**Priority:** P1
**Type:** Integration
**Package:** internal/extension
**Related Tasks:** 05

## Objective

Validate the full round-trip: an extension publishes resource records via `resources/snapshot`, then reads them back via `resources/list`. The extension must be able to see its own source-scoped records, confirming the snapshot-to-read path works end-to-end.

## Preconditions

- Real SQLite database via `t.TempDir()` with resource tables created
- Extension host initialized with a test extension that has been granted `resource_kinds=["tool"]`
- Extension has completed initialize and holds a valid `session_nonce`
- Test extension capable of issuing `resources/snapshot` and `resources/list` JSON-RPC calls

## Test Steps

1. Extension issues `resources/snapshot` with nonce and a payload of 3 tool records: `tool-a`, `tool-b`, `tool-c`.
   **Expected:** Snapshot accepted. No error returned.

2. Extension issues `resources/list` for `kind=tool`.
   **Expected:** Response contains exactly 3 records: `tool-a`, `tool-b`, `tool-c`. Each record has the correct `source` matching the extension's source identifier.

3. Verify each record's `data` payload matches what was submitted in the snapshot.
   **Expected:** Data fields are identical — no truncation, no mutation.

4. Extension issues a second `resources/snapshot` replacing `tool-b` with `tool-d` (snapshot contains `tool-a`, `tool-c`, `tool-d`).
   **Expected:** Snapshot accepted.

5. Extension issues `resources/list` for `kind=tool` again.
   **Expected:** Response contains exactly 3 records: `tool-a`, `tool-c`, `tool-d`. `tool-b` has been removed (snapshot is declarative/full-replace for that source+kind).

6. Verify `tool-b` no longer exists in the resource store.
   **Expected:** Direct query for `tool-b` returns no results.

## Edge Cases

- Empty snapshot (zero records) — removes all records for that source+kind
- Snapshot with duplicate IDs — last entry wins or error returned, never silent data loss
- Extension tries to list a kind it was not granted — returns error or empty set (no cross-kind leakage)
- Snapshot with very large data payload (e.g., 1MB JSON) — accepted if within limits, rejected cleanly if not
- Concurrent snapshots from the same extension — serialized correctly, no partial state
