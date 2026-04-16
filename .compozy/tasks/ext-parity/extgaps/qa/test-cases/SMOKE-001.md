# SMOKE-001: Full activation lifecycle

**Priority:** P0 (Critical)
**Type:** Smoke
**Component:** Full stack (API → Service → Store)

## Objective

Validate the happy-path lifecycle: Catalog → Preview → Activate → List → Get → Deactivate.

## Test Steps

1. GET `/api/bundles/catalog`
   **Expected:** Returns at least one bundle entry

2. POST `/api/bundles/preview` with valid request
   **Expected:** HTTP 200, preview contains activation with inventory

3. POST `/api/bundles/activations` with same request
   **Expected:** HTTP 201, activation created with same ID as preview

4. GET `/api/bundles/activations`
   **Expected:** HTTP 200, list contains the new activation

5. GET `/api/bundles/activations/{id}`
   **Expected:** HTTP 200, activation details match creation response

6. DELETE `/api/bundles/activations/{id}`
   **Expected:** HTTP 204

7. GET `/api/bundles/activations/{id}`
   **Expected:** HTTP 404
