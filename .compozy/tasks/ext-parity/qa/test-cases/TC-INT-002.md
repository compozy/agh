# TC-INT-002: Snapshot cannot overwrite daemon-owned record

**Priority:** P0
**Type:** Integration
**Package:** internal/resources
**Related Tasks:** 01

## Objective

Validate that the resource store enforces ownership boundaries — an extension snapshot targeting a `(kind, id)` pair already owned by the daemon must be rejected with a 409 conflict. This prevents extensions from silently corrupting daemon-managed state.

## Preconditions

- Real SQLite database via `t.TempDir()` with resource tables created
- Resource store initialized and ready to accept writes
- A daemon-owned record already persisted with a known `(kind, id)` pair (e.g., `kind=hook.binding`, `id=daemon-hook-1`, `source=daemon`)

## Test Steps

1. Insert a resource record with `source=daemon`, `kind=hook.binding`, `id=daemon-hook-1` via the daemon write path.
   **Expected:** Record persisted successfully. Readable via list/get.

2. Construct an extension snapshot payload targeting the same `kind=hook.binding`, `id=daemon-hook-1` but with `source=ext-alpha`.
   **Expected:** Payload is well-formed and accepted by the snapshot API shape.

3. Submit the extension snapshot.
   **Expected:** Operation returns a 409 conflict error (or equivalent ownership violation error). The error message identifies the conflicting record.

4. Read back the original record.
   **Expected:** Record is unchanged — `source` is still `daemon`, `data` payload is the original value, `updated_at` has not changed.

5. Submit an extension snapshot for a different `id` (e.g., `id=ext-hook-1`) with the same `kind`.
   **Expected:** Snapshot succeeds. Extension-owned record is created alongside the daemon-owned record.

## Edge Cases

- Extension tries to overwrite with identical data — still rejected (ownership check, not content check)
- Extension tries to delete a daemon-owned record via snapshot omission — daemon record must survive
- Two different extensions both try to claim the same `(kind, id)` — first writer wins, second gets 409
- Daemon overwrites its own record — succeeds (same source)
