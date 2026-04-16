# SMOKE-002: Primary channel binding lifecycle

**Priority:** P0 (Critical)
**Type:** Smoke
**Component:** Network settings flow

## Test Steps

1. GET `/api/bundles/network/settings`
   **Expected:** effective_default_source = "config"

2. POST `/api/bundles/activations` with `bind_primary_channel_as_default: true`
   **Expected:** HTTP 201

3. GET `/api/bundles/network/settings`
   **Expected:** effective_default_channel = profile's primary channel, effective_default_source = activation ID

4. DELETE `/api/bundles/activations/{id}`
   **Expected:** HTTP 204

5. GET `/api/bundles/network/settings`
   **Expected:** effective_default_source = "config" (fallback)
