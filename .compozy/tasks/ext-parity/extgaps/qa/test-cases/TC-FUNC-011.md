# TC-FUNC-011: Network settings returns configured and effective defaults

**Priority:** P1 (High)
**Type:** Functional
**Component:** `internal/bundles/service.go` — `Service.NetworkSettings()`

## Objective

Validate that the NetworkSettings endpoint correctly reports both configured and effective default channels, along with all declared channels from active bundles.

## Preconditions

- Bundle service initialized with `WithConfiguredDefaultChannel("default")`
- Extension with bundle containing 2 channels (primary + secondary)

## Test Steps

1. Query network settings with no activations
   **Expected:** ConfiguredDefaultChannel="default", EffectiveDefaultChannel="default", EffectiveDefaultSource="config", DeclaredChannels=[]

2. Activate bundle without primary binding
   **Expected:** Network settings show declared channels but effective default unchanged

3. Verify declared channels include activation context
   **Expected:** Each DeclaredChannel has ActivationID, ExtensionName, BundleName, ProfileName, WorkspaceID, Name, Description, Primary flag

4. Verify declared channels are sorted
   **Expected:** Sorted by (ExtensionName, BundleName, ProfileName, Name)

5. Activate with primary binding
   **Expected:** EffectiveDefaultChannel changes to primary channel name, EffectiveDefaultSource = activation ID

6. Query settings concurrently from multiple goroutines
   **Expected:** No data races (RWMutex protects settings)

## Edge Cases

- Service is nil → "bundles: service is required"
- Store is nil → "bundles: store is required"
- No configured default → defaults to "default" string
- Effective default is whitespace-only → treated as empty, falls back to configured
