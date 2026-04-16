# TC-FUNC-009: Primary channel binding sets effective default channel

**Priority:** P0 (Critical)
**Type:** Functional
**Component:** `internal/bundles/service.go` — `Reconcile()`, `NetworkSettings()`

## Objective

Validate that when an activation sets `BindPrimaryChannelAsDefault: true`, the effective default channel is updated to the profile's primary channel.

## Preconditions

- Extension with bundle containing profile with primary channel "my-channel"
- Configured default channel is "default"

## Test Steps

1. Check initial network settings
   **Expected:** ConfiguredDefaultChannel="default", EffectiveDefaultChannel="default", EffectiveDefaultSource="config"

2. Activate bundle with `BindPrimaryChannelAsDefault: true`
   **Expected:** Activation succeeds

3. Check network settings after activation
   **Expected:** ConfiguredDefaultChannel="default", EffectiveDefaultChannel="my-channel", EffectiveDefaultSource=activationID

4. Deactivate the bundle
   **Expected:** Deactivation succeeds

5. Check network settings after deactivation
   **Expected:** EffectiveDefaultChannel="default", EffectiveDefaultSource="config"

## Edge Cases

- Profile with empty primary channel and BindPrimaryChannelAsDefault=true → reconciliation error: "cannot bind an empty primary channel"
- Multiple activations without primary binding → effective default stays at config
- Activation claims primary, then update to unclaim → falls back to config
