# TC-FUNC-012: Failed reconciliation rolls back activation creation

**Priority:** P0 (Critical)
**Type:** Functional
**Component:** `internal/bundles/service.go` — `Service.Activate()` rollback path

## Objective

Validate that when reconciliation fails after creating a new activation, the activation is deleted (rolled back) and the error is propagated.

## Preconditions

- AutomationSyncer configured to return an error on SyncManagedDefinitions
- Extension with bundle containing jobs (to trigger automation sync)

## Test Steps

1. Configure automation syncer to fail with "sync failed"
   **Expected:** Syncer returns error

2. Call `Service.Activate(ctx, validRequest)`
   **Expected:** Returns error containing "sync failed"

3. Verify activation was rolled back
   **Expected:** `store.GetBundleActivation(ctx, expectedID)` returns `ErrActivationNotFound`

4. Verify no inventory was left behind
   **Expected:** Inventory for expected ID is empty

## Edge Cases

- Rollback itself fails (store.DeleteBundleActivation errors) → original reconciliation error still returned
- Bridge sync fails → same rollback behavior
- Multiple errors during reconciliation → errors.Join combines them
