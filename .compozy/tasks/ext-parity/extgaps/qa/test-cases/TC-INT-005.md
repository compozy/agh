# TC-INT-005: HTTP GET /api/bundles/network/settings returns channel state

**Priority:** P1 (High)
**Type:** Integration
**Component:** `internal/api/core/bundles.go` — `BundleNetworkSettings`

## Objective

Validate the network settings endpoint returns correct configured/effective defaults and declared channels.

## Preconditions

- Bundle service initialized
- Known configured default channel

## Test Steps

1. GET `/api/bundles/network/settings` with no activations
   **Expected:** HTTP 200, `network.configured_default_channel` = config value, `network.effective_default_channel` = config value, `network.effective_default_source` = "config"

2. Activate bundle with primary binding, then GET
   **Expected:** effective_default_channel = primary channel name, effective_default_source = activation ID, declared_channels populated

3. Verify declared_channels payload structure
   **Expected:** Each item has activation_id, extension_name, bundle_name, profile_name, workspace_id, name, description, primary flag

4. GET when bundle service is nil
   **Expected:** HTTP 503

## Edge Cases

- Concurrent reads during reconciliation → RWMutex ensures consistent snapshot
