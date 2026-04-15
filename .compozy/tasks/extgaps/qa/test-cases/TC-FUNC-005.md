# TC-FUNC-005: Activation update modifies primary channel binding

**Priority:** P1 (High)
**Type:** Functional
**Component:** `internal/bundles/service.go` — `Service.UpdateActivation()`

## Objective

Validate that updating an activation's `BindPrimaryChannelAsDefault` flag works correctly, triggers reconciliation, and updates network settings.

## Preconditions

- Active bundle activation exists (from TC-FUNC-004)
- Bundle has a primary channel declared

## Test Steps

1. Activate bundle with `BindPrimaryChannelAsDefault: false`
   **Expected:** Activation created. Network settings show effective default from config.

2. Call `Service.UpdateActivation(ctx, UpdateActivationRequest{ID: activationID, BindPrimaryChannelAsDefault: true})`
   **Expected:** Returns updated activation preview with binding enabled

3. Verify network settings changed
   **Expected:** EffectiveDefaultChannel = profile's primary channel, EffectiveDefaultSource = activation ID

4. Call `Service.UpdateActivation(ctx, UpdateActivationRequest{ID: activationID, BindPrimaryChannelAsDefault: false})`
   **Expected:** Effective default falls back to config

5. Verify UpdatedAt timestamp changed
   **Expected:** UpdatedAt > previous UpdatedAt

## Edge Cases

- Update non-existent activation → `ErrActivationNotFound` (404)
- Update to claim primary when another activation already claims it → `ErrDefaultChannelBusy` (409)
- Reconciliation failure during update → reverts to previous state
