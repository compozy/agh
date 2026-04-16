# TC-FUNC-003: Activation preview returns materialized resources without persisting

**Priority:** P1 (High)
**Type:** Functional
**Component:** `internal/bundles/service.go` — `Service.PreviewActivation()`

## Objective

Validate that preview returns the same activation preview as actual activation, but does NOT persist anything to the database.

## Preconditions

- Extension "test-ext" installed with bundle "notify" containing profile "default" with 1 job, 1 trigger, 1 bridge
- Bundle service initialized

## Test Steps

1. Call `Service.PreviewActivation(ctx, ActivateRequest{ExtensionName: "test-ext", BundleName: "notify", ProfileName: "default", Scope: "global"})`
   **Expected:** Returns `ActivationPreview` with populated Activation, Bundle, Profile, and Inventory

2. Verify preview.Activation.ID is a stable hash-based ID
   **Expected:** ID starts with "act\_" and is deterministic for same inputs

3. Verify preview.Inventory contains 3 items (1 job, 1 trigger, 1 bridge)
   **Expected:** Each item has correct ResourceKind, ResourceID, ResourceName

4. Call `Service.ListActivations(ctx)`
   **Expected:** Returns empty list (preview did NOT persist)

5. Call `store.GetBundleActivation(ctx, preview.Activation.ID)`
   **Expected:** Returns `ErrActivationNotFound`

6. Verify no automation sync was called
   **Expected:** AutomationSyncer was NOT invoked

## Edge Cases

- Preview with non-existent extension → `ErrExtensionNotFound` (404)
- Preview with non-existent bundle → `ErrBundleNotFound` (404)
- Preview with non-existent profile → `ErrProfileNotFound` (404)
- Preview with workspace scope and missing workspace → validation error
- Preview with webhook trigger in bundle → `ErrWebhookUnsupported` (400)
