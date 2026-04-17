# TC-INT-006: Extension disable blocked when bundles active (HTTP 409)

**Priority:** P0 (Critical)
**Type:** Integration
**Component:** `internal/extension/registry.go` — Disable/Uninstall guards

## Objective

Validate that attempting to disable or uninstall an extension with active bundle activations returns 409 conflict.

## Preconditions

- Extension "test-ext" installed with bundle
- Bundle activated for "test-ext"

## Test Steps

1. Attempt to disable "test-ext" via API
   **Expected:** HTTP 409, error includes `ErrExtensionHasActiveBundles`

2. Attempt to uninstall "test-ext" via API
   **Expected:** HTTP 409, error includes `ErrExtensionHasActiveBundles`

3. Deactivate all bundles for "test-ext"
   **Expected:** Deactivation succeeds

4. Retry disable "test-ext"
   **Expected:** Now succeeds

5. Retry uninstall "test-ext"
   **Expected:** Now succeeds

## Edge Cases

- Extension with multiple activations → all must be deactivated before disable
- Extension with workspace-scoped activations → same blocking behavior
