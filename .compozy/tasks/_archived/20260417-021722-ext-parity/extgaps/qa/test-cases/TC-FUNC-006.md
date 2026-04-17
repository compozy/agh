# TC-FUNC-006: Deactivation removes activation and cleans up resources

**Priority:** P0 (Critical)
**Type:** Functional
**Component:** `internal/bundles/service.go` — `Service.Deactivate()`

## Objective

Validate that deactivating a bundle removes the activation from the store, triggers reconciliation to clean up managed resources, and updates network settings.

## Preconditions

- Active bundle activation with materialized jobs, triggers, and bridges
- Primary channel binding active

## Test Steps

1. Confirm activation exists and resources are materialized
   **Expected:** Activation retrievable, inventory has items

2. Call `Service.Deactivate(ctx, activationID)`
   **Expected:** Returns nil error

3. Verify activation removed from store
   **Expected:** `store.GetBundleActivation(ctx, activationID)` returns `ErrActivationNotFound`

4. Verify reconciliation cleaned up automation resources
   **Expected:** SyncManagedDefinitions called with empty desired lists (or without the deactivated resources)

5. Verify package-owned bridge instances deleted
   **Expected:** Bridge instances with Source=BridgeInstanceSourcePackage removed

6. Verify network settings updated
   **Expected:** If this was the only activation claiming primary channel, effective default falls back to config

7. Verify inventory cascade-deleted
   **Expected:** `store.ListBundleActivationInventory(ctx, activationID)` returns empty

## Edge Cases

- Deactivate non-existent activation → `ErrActivationNotFound`
- Deactivate with whitespace-padded ID → trimmed before lookup
- Reconciliation failure after delete → activation re-created (rollback)
- Deactivate last activation for an extension → extension can now be disabled
