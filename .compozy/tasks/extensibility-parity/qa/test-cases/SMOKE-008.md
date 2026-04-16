# SMOKE-008: Bundle Activation Creates Owned Resources

**Priority:** P0
**Type:** Smoke
**Package:** internal/bundles
**Related Tasks:** 12

## Objective

Validate that creating a bundle and a corresponding bundle.activation resource triggers activation fan-out that automatically creates owned child resources (automation.job, bridge.instance) with correct owner_kind and owner_id pointing back to the bundle activation. This confirms the bundle lifecycle correctly composes multiple resource kinds through the activation fan-out mechanism.

## Preconditions

- Resource store initialized with kind codecs for "bundle", "bundle.activation", "automation.job", and "bridge.instance"
- Reconcile driver configured with the bundles projector and all dependent projectors
- Bundle spec includes declarations for one automation job and one bridge instance

## Test Steps

1. **Create a bundle resource** with kind="bundle", spec containing:
   - name="observability-pack"
   - automations: one automation job declaration (name="log-rotate", trigger="cron", schedule="@hourly")
   - bridges: one bridge instance declaration (name="slack-alerts", type="slack", config with channel="#ops")
   **Expected:** Bundle resource is persisted with version=1. No child resources exist yet.

2. **Create a bundle.activation resource** with kind="bundle.activation", spec containing bundle_id referencing the bundle from step 1, scope="workspace".
   **Expected:** bundle.activation resource is persisted with version=1.

3. **Trigger reconciliation** so the bundles projector processes the activation.
   **Expected:** Reconciliation completes without error. Fan-out creates child resources.

4. **List resources with kind="automation.job"** filtered by owner_kind="bundle.activation" and owner_id matching the activation resource ID.
   **Expected:** Exactly one automation.job resource exists with name="log-rotate", trigger="cron", schedule="@hourly". Its owner_kind="bundle.activation" and owner_id matches the activation ID from step 2.

5. **List resources with kind="bridge.instance"** filtered by owner_kind="bundle.activation" and owner_id matching the activation resource ID.
   **Expected:** Exactly one bridge.instance resource exists with name="slack-alerts", type="slack". Its owner_kind="bundle.activation" and owner_id matches the activation ID from step 2.

6. **Delete the bundle.activation resource** and trigger reconciliation.
   **Expected:** All owned child resources (the automation.job and bridge.instance) are cascade-deleted. Listing by the old owner returns empty results.

7. **Verify the bundle resource itself still exists** after activation deletion.
   **Expected:** The bundle resource at kind="bundle" is unaffected by the activation deletion. It remains at its original version.

## Edge Cases

- Activating the same bundle twice creates two independent sets of owned resources with different activation IDs
- A bundle with zero automations and zero bridges activates successfully but creates no child resources
- A bundle activation referencing a non-existent bundle_id fails reconciliation with a clear error
- Updating a bundle spec after activation does not retroactively update already-created child resources (activation is a point-in-time snapshot)
- Deleting the bundle resource while an activation exists: activation and its children remain (no cascading from bundle to activation)
- A bundle with many declarations (10+ automations, 10+ bridges) fans out all children in a single reconciliation pass
- Concurrent activations of the same bundle do not create duplicate children within a single activation (each activation's children are isolated)
- Child resources created by fan-out have source="bundle.activation" to distinguish them from manually created resources
