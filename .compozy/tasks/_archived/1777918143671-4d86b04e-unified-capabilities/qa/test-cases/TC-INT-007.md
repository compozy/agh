# TC-INT-007: Bundle Network Defaults In Status

**Priority:** P2
**Type:** Integration
**Systems:** Bundle Service, API Core, Settings
**API Endpoint:** `GET /api/network/status`

## Objective

Verify bundle-declared network defaults flow into the runtime status payload visible to web and CLI consumers.

## Preconditions

- Bundle service has activated bundles with declared channels and default-channel configuration.
- Network runtime is enabled.

## Test Steps

1. Activate a bundle that declares network channels.
   **Expected:** Bundle service reports declared channels and effective default source.
2. Request network status.
   **Expected:** Status payload includes declared channels and configured/effective defaults.
3. Disable or remove the bundle.
   **Expected:** Status payload reflects the updated effective default.

## Edge Cases

| Variation | Input | Expected Result |
| --- | --- | --- |
| Bundle settings error | service error | status returns 500 with context |
| No bundles service | nil service | status still returns runtime payload |
| Conflicting declarations | duplicate channel names | deterministic bundle resolution |

## Related

- TC-FUNC-001
