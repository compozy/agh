# SMOKE-003: Extension lifecycle guard

**Priority:** P0 (Critical)
**Type:** Smoke
**Component:** Extension registry + bundle activation

## Test Steps

1. Install extension with bundle
   **Expected:** Extension active

2. Activate bundle for extension
   **Expected:** HTTP 201

3. Attempt to disable extension
   **Expected:** HTTP 409 with ErrExtensionHasActiveBundles

4. Deactivate all bundles for extension
   **Expected:** HTTP 204

5. Retry disable extension
   **Expected:** Succeeds
