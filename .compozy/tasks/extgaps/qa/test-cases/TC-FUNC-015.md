# TC-FUNC-015: Bridge materialization resolves platform from provider extension

**Priority:** P1 (High)
**Type:** Functional
**Component:** `internal/bundles/service.go` — `Service.materializeBridge()`

## Objective

Validate that bridge materialization correctly resolves the platform field from the appropriate extension when not explicitly declared in the bridge preset.

## Preconditions

- "owner-ext" installed with manifest declaring `bridge.platform = "slack"`
- "provider-ext" installed with manifest declaring `bridge.platform = "discord"`
- Bundle on "owner-ext" with bridge presets

## Test Steps

### Case 1: Platform explicitly set in preset

1. Bridge preset has `Platform: "telegram"`
   **Expected:** Bridge instance platform = "telegram" (uses explicit value)

### Case 2: Platform from owner extension

2. Bridge preset has empty Platform, empty ExtensionName (inherits activation's extension)
   **Expected:** Platform resolved from owner-ext's manifest.bridge.platform = "slack"

### Case 3: Platform from different provider extension

3. Bridge preset has `ExtensionName: "provider-ext"`, empty Platform
   **Expected:** Platform resolved from provider-ext's manifest.bridge.platform = "discord"

### Case 4: Provider extension unavailable

4. Bridge preset has `ExtensionName: "missing-ext"`, empty Platform
   **Expected:** Error: "bridge provider missing-ext is unavailable"

### Common validations

5. Verify bridge instance has `Source: BridgeInstanceSourcePackage`
   **Expected:** Source field correctly set

6. Verify bridge instance has `Enabled: false`, `Status: BridgeStatusDisabled`
   **Expected:** Bridges start disabled, not auto-enabled

7. Verify `instance.Validate()` is called
   **Expected:** Invalid bridge (e.g., empty DisplayName) causes error

## Edge Cases

- Owner extension has nil manifest → uses loader fallback path
- Provider extension returns nil → "is unavailable" error
- ExtensionName is same as activation extension (case-insensitive) → uses owner manifest directly (no extra load)
