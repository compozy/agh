# TC-INT-004: HTTP DELETE /api/bundles/activations/:id deactivates

**Priority:** P0 (Critical)
**Type:** Integration
**Component:** `internal/api/core/bundles.go` — `DeleteBundleActivation`

## Objective

Validate the DELETE endpoint correctly removes an activation and triggers cleanup.

## Preconditions

- Active bundle activation exists

## Test Steps

1. DELETE `/api/bundles/activations/{id}`
   **Expected:** HTTP 204, no response body

2. GET `/api/bundles/activations/{id}` after delete
   **Expected:** HTTP 404

3. DELETE with non-existent ID
   **Expected:** HTTP 404

4. DELETE when bundle service is nil
   **Expected:** HTTP 503

## Edge Cases

- Double-delete same ID → first returns 204, second returns 404
- ID with whitespace → trimmed
