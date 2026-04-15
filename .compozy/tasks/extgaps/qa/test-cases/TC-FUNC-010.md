# TC-FUNC-010: Second primary channel claim returns 409 conflict

**Priority:** P0 (Critical)
**Type:** Functional
**Component:** `internal/bundles/service.go` — `validatePrimaryChannelClaim()`

## Objective

Validate that only one activation at a time can claim the effective default channel via `BindPrimaryChannelAsDefault`, and a second attempt returns an appropriate conflict error.

## Preconditions

- Extension with two bundles/profiles, each with a primary channel
- Bundle service initialized

## Test Steps

1. Activate bundle-A/profile-A with `BindPrimaryChannelAsDefault: true`
   **Expected:** Activation succeeds

2. Activate bundle-B/profile-B with `BindPrimaryChannelAsDefault: true`
   **Expected:** Error wrapping `ErrDefaultChannelBusy`

3. Verify error message includes the ID of the existing claimant
   **Expected:** Error contains first activation's ID

4. Verify HTTP status code mapping
   **Expected:** `StatusForBundleError(ErrDefaultChannelBusy)` returns 409

5. Deactivate bundle-A
   **Expected:** Deactivation succeeds

6. Retry activate bundle-B with `BindPrimaryChannelAsDefault: true`
   **Expected:** Now succeeds (claim is free)

## Edge Cases

- Same activation updating itself from false→true when it's already the only claimant → succeeds (self-replace)
- Two activations both with BindPrimaryChannelAsDefault=false → no conflict
