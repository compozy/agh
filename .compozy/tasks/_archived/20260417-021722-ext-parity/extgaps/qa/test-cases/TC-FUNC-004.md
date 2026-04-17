# TC-FUNC-004: Activation creates and persists all resources with inventory

**Priority:** P0 (Critical)
**Type:** Functional
**Component:** `internal/bundles/service.go` — `Service.Activate()`

## Objective

Validate that activating a bundle persists the activation, triggers reconciliation, materializes all resources (jobs, triggers, bridges), and records inventory.

## Preconditions

- Extension "test-ext" with bundle "analytics" containing profile "full" with:
  - 2 jobs (daily-report, cleanup)
  - 1 trigger (on-session-end)
  - 1 bridge (slack-notifier)
  - 2 channels (primary: "analytics-main", secondary: "analytics-debug")
- AutomationSyncer and Store mocked/real

## Test Steps

1. Call `Service.Activate(ctx, ActivateRequest{ExtensionName: "test-ext", BundleName: "analytics", ProfileName: "full", Scope: "global", BindPrimaryChannelAsDefault: false})`
   **Expected:** Returns ActivationPreview with activation.ID populated

2. Verify activation persisted in store
   **Expected:** `store.GetBundleActivation(ctx, activation.ID)` returns the activation

3. Verify activation fields
   **Expected:** ExtensionName="test-ext", BundleName="analytics", ProfileName="full", Scope="global", WorkspaceID="", CreatedAt set, UpdatedAt set

4. Verify automation syncer was called with `JobSourcePackage`
   **Expected:** SyncManagedDefinitions called with 2 desired jobs and 1 desired trigger

5. Verify job IDs are stable hashes
   **Expected:** job IDs start with "job\_" and are deterministic

6. Verify trigger IDs are stable hashes
   **Expected:** trigger IDs start with "trg\_"

7. Verify bridge instance was inserted into store
   **Expected:** Bridge with ID starting "bri\_", Source=BridgeInstanceSourcePackage, Enabled=false, Status=BridgeStatusDisabled

8. Verify inventory contains 4 items (2 jobs + 1 trigger + 1 bridge)
   **Expected:** `store.ListBundleActivationInventory(ctx, activation.ID)` returns 4 items with correct kinds

9. Re-activate same bundle (idempotent upsert)
   **Expected:** Same activation ID returned, CreatedAt preserved from first activation, UpdatedAt updated

## Edge Cases

- Activation with same inputs produces identical ID (idempotent)
- Activation with whitespace-padded names → trimmed before hashing
- Store failure during creation → error returned, no partial state
- Reconciliation failure → activation rolled back (deleted if new, reverted if existing)
