# TC-INT-017: Activation creates owned automation and bridge records

**Priority:** P0
**Type:** Integration
**Package:** internal/bundles
**Related Tasks:** 12

## Objective

Validate that activating a bundle creates owned `automation.job` and `bridge.instance` resource records with correct `owner_kind` and `owner_id` fields. Bundle activation is the composition primitive — it declares a set of resources that belong to the bundle and must be managed as a unit.

## Preconditions

- Real SQLite database via `t.TempDir()` with resource tables created
- Resource store initialized with projectors for `automation.job` and `bridge.instance`
- Bundle manager/runtime initialized
- A test bundle definition that declares 2 automation jobs and 1 bridge instance

## Test Steps

1. Define a test bundle `bundle-alpha` that declares:
   - `automation.job` with `id=bundle-auto-1` (a scheduled task)
   - `automation.job` with `id=bundle-auto-2` (an event-triggered task)
   - `bridge.instance` with `id=bundle-bridge-1` (a WebSocket bridge)
   **Expected:** Bundle definition is valid and loadable.

2. Activate `bundle-alpha`.
   **Expected:** Activation completes without error.

3. Query `resource_records` for records with `owner_kind=bundle` and `owner_id=bundle-alpha`.
   **Expected:** Exactly 3 records returned: `bundle-auto-1`, `bundle-auto-2`, `bundle-bridge-1`.

4. Verify `bundle-auto-1` has `kind=automation.job`, correct data payload, and `owner_kind=bundle`, `owner_id=bundle-alpha`.
   **Expected:** All fields correct.

5. Verify `bundle-auto-2` has `kind=automation.job`, correct data payload, and matching owner fields.
   **Expected:** All fields correct.

6. Verify `bundle-bridge-1` has `kind=bridge.instance`, correct data payload, and matching owner fields.
   **Expected:** All fields correct.

7. Verify the automation runtime has both jobs active and the bridge registry has the bridge instance.
   **Expected:** Projectors have reconciled. Runtime state reflects the 3 new resources.

8. Activate `bundle-alpha` again (idempotent re-activation).
   **Expected:** No duplicate records. Existing records may be updated but count remains 3.

## Edge Cases

- Bundle with zero resources — activation succeeds, no records created
- Bundle activation when one of the declared resource IDs already exists (owned by a different source) — 409 conflict, activation fails atomically
- Bundle definition with invalid resource data — activation fails before creating any records (atomic)
- Two bundles declaring resources with the same ID — second activation gets conflict
- Owner fields are queryable — can efficiently find all resources owned by a specific bundle
