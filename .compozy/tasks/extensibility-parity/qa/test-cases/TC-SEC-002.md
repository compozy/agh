# TC-SEC-002: Snapshot Cannot Overwrite Foreign-Source Record

**Priority:** P0
**Type:** Security
**Package:** internal/resources
**Related Tasks:** 01

## Objective

Validate that a snapshot operation targeting a `(kind, id)` pair already owned by a different source results in a 409 Conflict error, not a silent overwrite. Source ownership of records is immutable once established.

## Preconditions

- Extension `ext-alpha` has an active session with a valid nonce.
- Extension `ext-beta` has an active session with a valid nonce.
- `ext-alpha` has previously published a record `(tool, shared-tool-1)` via snapshot.

## Test Steps

1. As `ext-beta`, submit a snapshot that includes a record with `kind=tool` and `id=shared-tool-1` (the same key owned by `ext-alpha`).
   **Expected:** The snapshot is rejected with a 409 Conflict error. The error message indicates a source ownership conflict for the specific `(kind, id)` pair.

2. Query the record `(tool, shared-tool-1)` via an operator/admin read path.
   **Expected:** The record still contains `ext-alpha`'s original payload. No fields have been modified by `ext-beta`'s attempted snapshot.

3. As `ext-alpha`, submit a snapshot updating `(tool, shared-tool-1)` with new content.
   **Expected:** The update succeeds. The owning source can still modify its own records.

4. As `ext-beta`, submit a snapshot with a record using the same `kind=tool` but a unique `id=beta-tool-1`.
   **Expected:** The snapshot succeeds. `ext-beta` can create new records with unique IDs without interference.

5. As `ext-beta`, submit a snapshot containing both a valid new record and the conflicting `(tool, shared-tool-1)`.
   **Expected:** The entire snapshot is rejected atomically. The valid new record is NOT persisted. No partial application occurs.

## Edge Cases

- Extension attempts to delete and re-create a record with the same `(kind, id)` to claim ownership.
- Extension submits a snapshot where the `source` field in the record payload is manually set to the other extension's source identifier.
- Race condition: two extensions simultaneously attempt to claim the same previously-unowned `(kind, id)` -- exactly one must win, the other must get 409.
- Extension attempts to overwrite after the owning extension has been unregistered but its records have not been garbage-collected.

## Threat Model

This test prevents **resource hijacking via snapshot injection**. If an extension could silently overwrite records owned by another source, it could replace legitimate tool definitions with malicious ones, alter hook bindings to redirect control flow, or corrupt configuration data. The 409 Conflict guarantee ensures that source ownership is a hard boundary that cannot be bypassed through the snapshot API.
