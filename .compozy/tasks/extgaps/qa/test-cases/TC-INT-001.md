# TC-INT-001: HTTP POST /api/bundles/activations creates activation

**Priority:** P0 (Critical)
**Type:** Integration
**Component:** `internal/api/httpapi/` + `internal/api/core/bundles.go`

## Objective

Validate the full HTTP request→handler→service→store→response flow for bundle activation creation.

## Preconditions

- HTTP server running with test configuration
- Extension installed with bundle and profile
- Bundle service wired into RuntimeDeps

## Test Steps

1. POST `/api/bundles/activations` with valid JSON body:
   ```json
   {
     "extension_name": "test-ext",
     "bundle_name": "notify",
     "profile_name": "default",
     "scope": "global",
     "bind_primary_channel_as_default": false
   }
   ```
   **Expected:** HTTP 201, response body contains `activation` object with ID, timestamps, inventory

2. Verify response JSON structure matches `contract.BundleActivationResponse`
   **Expected:** Fields: id, extension_name, bundle_name, profile_name, scope, channels, jobs, triggers, bridges, inventory, created_at, updated_at

3. POST same request again (idempotent upsert)
   **Expected:** HTTP 201, same activation ID

4. POST with missing extension_name
   **Expected:** HTTP 400

5. POST with non-existent extension
   **Expected:** HTTP 404

6. POST with non-existent bundle
   **Expected:** HTTP 404

7. POST with invalid JSON body
   **Expected:** HTTP 400

8. POST when bundle service is nil
   **Expected:** HTTP 503

## Edge Cases

- Content-Type not application/json → 400
- Empty request body → 400
- Whitespace-only field values → treated as missing
