# TC-INT-018: Deactivation removes only owned records

**Priority:** P1
**Type:** Integration
**Package:** internal/bundles
**Related Tasks:** 12

## Objective

Validate that deactivating a bundle removes only the resource records owned by that bundle. Pre-existing automation and bridge records from other sources or bundles must remain untouched. This ensures bundle lifecycle isolation — bundles are self-contained units that do not interfere with each other or with operator-created resources.

## Preconditions

- Real SQLite database via `t.TempDir()` with resource tables created
- Resource store initialized with projectors for `automation.job` and `bridge.instance`
- Bundle manager/runtime initialized
- `bundle-alpha` activated with 2 automation jobs and 1 bridge (from TC-INT-017 setup)
- Additional pre-existing records from other sources already in the store

## Test Steps

1. Create pre-existing resources not owned by any bundle:
   - `automation.job` with `id=operator-auto-1`, `source=operator` (no owner fields or `owner_kind=operator`)
   - `bridge.instance` with `id=ext-bridge-1`, `source=ext-beta`
   **Expected:** Both records persisted.

2. Activate `bundle-alpha` creating its 3 owned records (as in TC-INT-017).
   **Expected:** Store contains 5 total records: 2 pre-existing + 3 bundle-owned.

3. Verify all 5 records are present.
   **Expected:** `operator-auto-1`, `ext-bridge-1`, `bundle-auto-1`, `bundle-auto-2`, `bundle-bridge-1` all exist.

4. Deactivate `bundle-alpha`.
   **Expected:** Deactivation completes without error.

5. Query `resource_records` for records with `owner_kind=bundle` and `owner_id=bundle-alpha`.
   **Expected:** Zero records returned. All 3 bundle-owned records are gone.

6. Query for `operator-auto-1`.
   **Expected:** Record still present and unchanged.

7. Query for `ext-bridge-1`.
   **Expected:** Record still present and unchanged.

8. Verify the automation runtime no longer has `bundle-auto-1` and `bundle-auto-2`.
   **Expected:** Projectors reconciled. Only `operator-auto-1` remains in the automation runtime.

9. Verify the bridge registry no longer has `bundle-bridge-1`.
   **Expected:** Only `ext-bridge-1` remains in the bridge registry.

10. Deactivate `bundle-alpha` again (idempotent deactivation).
    **Expected:** No error. No records affected.

## Edge Cases

- Deactivation during active automation execution — running job completes or is cancelled, record still removed
- Deactivation when one owned record was already manually deleted — remaining records still cleaned up, no error
- Two bundles with overlapping resource kinds but different IDs — deactivating one does not affect the other's records
- Deactivation with a database write error mid-transaction — atomic rollback, all records preserved
- After deactivation, re-activation creates fresh records — no ghost state from previous activation
