# TC-FUNC-001: API Network Status Payload

**Priority:** P0
**Type:** Functional
**Module:** API Core
**Requirement:** Network status contract remains stable across enabled, disabled, and bundle-default states.

## Objective

Verify `NetworkStatus` returns accurate runtime status and safely reports disabled mode without requiring a network service.

## Preconditions

- API core handlers are constructed with controlled config.
- Runtime status fixture includes kind metrics and disconnect metadata.
- Bundle service can optionally provide network default-channel settings.

## Test Steps

1. Call `GET /api/network/status` with `network.enabled=false`.
   **Expected:** 200 response, `enabled=false`, `status="disabled"`, and no dependency on `NetworkService`.
2. Call `GET /api/network/status` with `network.enabled=true` and a populated `network.Status`.
   **Expected:** 200 response mirrors listener, peer, channel, queue, sent, received, rejected, delivered, workflow, handoff, and kind metric values.
3. Inject bundle network settings.
   **Expected:** Status payload includes configured default channel, effective default channel, source, and declared channels.
4. Make bundle settings return an error.
   **Expected:** Handler returns 500 with wrapped bundle-settings context.

## Edge Cases

| Variation | Input | Expected Result |
| --- | --- | --- |
| Nil runtime status | `Network.Status` returns nil | 500 `network status is required` |
| Missing runtime when enabled | nil service | 500 for status, 503 for service-required endpoints |
| Empty last disconnect | whitespace string | trimmed to empty string |

## Related

- SMOKE-001
- TC-INT-006
