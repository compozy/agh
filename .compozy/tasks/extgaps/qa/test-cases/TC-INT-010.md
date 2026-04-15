# TC-INT-010: All HTTP error codes match StatusForBundleError mapping

**Priority:** P1 (High)
**Type:** Integration
**Component:** `internal/api/core/bundles.go` — `StatusForBundleError()`

## Objective

Validate that the error-to-HTTP-status mapping function returns correct status codes for all known bundle errors.

## Preconditions

None (pure function test)

## Test Steps

1. `StatusForBundleError(nil)` → 200
   **Expected:** HTTP 200 OK

2. `StatusForBundleError(ErrActivationNotFound)` → 404
   **Expected:** HTTP 404 Not Found

3. `StatusForBundleError(ErrBundleNotFound)` → 404
   **Expected:** HTTP 404 Not Found

4. `StatusForBundleError(ErrProfileNotFound)` → 404
   **Expected:** HTTP 404 Not Found

5. `StatusForBundleError(extensionpkg.ErrExtensionNotFound)` → 404
   **Expected:** HTTP 404 Not Found

6. `StatusForBundleError(ErrDefaultChannelBusy)` → 409
   **Expected:** HTTP 409 Conflict

7. `StatusForBundleError(extensionpkg.ErrExtensionHasActiveBundles)` → 409
   **Expected:** HTTP 409 Conflict

8. `StatusForBundleError(ErrWebhookUnsupported)` → 400
   **Expected:** HTTP 400 Bad Request

9. `StatusForBundleError(workspacepkg.ErrWorkspaceNotFound)` → delegates to StatusForWorkspaceError
   **Expected:** 404

10. `StatusForBundleError(errors.New("unknown"))` → 400
    **Expected:** HTTP 400 Bad Request (default)

11. Wrapped errors: `StatusForBundleError(fmt.Errorf("context: %w", ErrActivationNotFound))` → 404
    **Expected:** errors.Is traversal works through wrapping

## Edge Cases

- Errors joined with errors.Join → first matching error determines status
