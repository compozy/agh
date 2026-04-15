# TC-INT-003: HTTP PATCH /api/bundles/activations/:id updates binding

**Priority:** P1 (High)
**Type:** Integration
**Component:** `internal/api/core/bundles.go` — `UpdateBundleActivation`

## Objective

Validate the PATCH endpoint correctly updates the primary channel binding flag.

## Preconditions

- Active bundle activation exists

## Test Steps

1. PATCH `/api/bundles/activations/{id}` with `{"bind_primary_channel_as_default": true}`
   **Expected:** HTTP 200, activation returned with binding enabled

2. PATCH with non-existent ID
   **Expected:** HTTP 404

3. PATCH when another activation already claims primary
   **Expected:** HTTP 409

4. PATCH with invalid JSON
   **Expected:** HTTP 400

5. PATCH when bundle service is nil
   **Expected:** HTTP 503

## Edge Cases

- ID with whitespace → trimmed by handler
- Boolean default value (false) when field omitted from JSON → binding disabled
