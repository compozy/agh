# TC-FUNC-013: Failed reconciliation rolls back activation update

**Priority:** P1 (High)
**Type:** Functional
**Component:** `internal/bundles/service.go` — `Service.UpdateActivation()` and `Service.Deactivate()` rollback

## Objective

Validate that when reconciliation fails after updating or deactivating an existing activation, the previous state is restored.

## Preconditions

- Active bundle activation exists
- AutomationSyncer can be switched to fail mode

## Test Steps

### Update rollback

1. Activate bundle with `BindPrimaryChannelAsDefault: false`
   **Expected:** Activation created successfully

2. Configure automation syncer to fail

3. Call `Service.UpdateActivation(ctx, {ID: activationID, BindPrimaryChannelAsDefault: true})`
   **Expected:** Returns reconciliation error

4. Verify activation reverted to previous state
   **Expected:** `store.GetBundleActivation(ctx, activationID)` shows BindPrimaryChannelAsDefault=false

### Deactivate rollback

5. Configure automation syncer to fail

6. Call `Service.Deactivate(ctx, activationID)`
   **Expected:** Returns reconciliation error

7. Verify activation was re-created
   **Expected:** `store.GetBundleActivation(ctx, activationID)` succeeds (activation restored)

## Edge Cases

- Rollback store operation fails → original error still returned
- Activation state is exactly restored (all fields including timestamps)
