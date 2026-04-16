# TC-FUNC-030: Activation fan-out does not create reverse dependency cycle

**Priority:** P2
**Type:** Functional
**Package:** internal/bundles
**Related Tasks:** 12

## Objective

Validate that when a bundle.activation fans out to create owned automation.* and bridge.instance records through the resource store, those records are processed independently by their respective projectors without creating a reverse dependency (DependsOn back-edge) from the child kinds to the bundle kind. The dependency graph must be strictly one-directional: bundle.activation depends on the kinds it creates, but those kinds must not depend on bundle.activation. This prevents circular reconfiguration loops and ensures projector ordering remains acyclic.

## Preconditions

- Resource runtime is active with projectors registered for: bundles, automation, bridges.
- A bundle "infra-bundle" exists with allowlist: automation.job, automation.trigger, bridge.instance.
- The projector dependency graph is inspectable (via test helper or internal API).
- No existing activation or owned records for this bundle.

## Test Steps

1. Inspect the projector dependency graph before any activation.
   **Expected:** The bundles projector may declare dependencies on other projectors (or none), but automation and bridges projectors do NOT declare a dependency on the bundles projector. The graph is a DAG with no cycles involving bundles.

2. Create a bundle.activation for "infra-bundle" that fans out to create: automation.job "sync-job", automation.trigger "sync-trigger", bridge.instance "sync-bridge".
   **Expected:** Activation succeeds. Three owned records are created in the resource store.

3. Trigger a full projector reconciliation cycle (all projectors run Build + Apply).
   **Expected:** Reconciliation completes without deadlock or infinite loop. Each projector processes its records independently.

4. Inspect the projector dependency graph after activation.
   **Expected:** Graph is unchanged from step 1. Creating owned records did not inject new DependsOn edges. The automation projector still does not depend on the bundles projector. The bridges projector still does not depend on the bundles projector.

5. Verify that the automation projector processed "sync-job" and "sync-trigger" independently.
   **Expected:** The automation scheduler has "sync-job" registered and "sync-trigger" wired. These were picked up from the resource store by the automation projector scanning for automation.* kinds, not by a callback from the bundles projector.

6. Verify that the bridges projector processed "sync-bridge" independently.
   **Expected:** The bridge registry has "sync-bridge" with appropriate connection state. It was picked up by the bridges projector scanning for bridge.instance kinds, not via a bundles projector notification.

7. Delete the bundle.activation. Run another reconciliation cycle.
   **Expected:** Owner-indexed cleanup removes "sync-job", "sync-trigger", "sync-bridge" from the store. The automation and bridges projectors pick up the deletions in their next Build and remove the entries from their live state. No reverse notification from automation/bridges back to bundles occurs.

8. Verify the projector dependency graph after deletion.
   **Expected:** Graph remains unchanged. No transient edges were added or removed during the lifecycle.

## Edge Cases

- Bundle activation that creates a record of kind bundle (nested bundles): verify whether this is blocked by the allowlist or if it is permitted but still does not create a cycle (bundle projector processes all bundle kinds, not a separate projector per bundle).
- Automation projector Build running concurrently with bundle activation fan-out: the automation projector sees a consistent snapshot (either the records exist or they don't), no partial writes.
- Bridge projector Apply failing for a bundle-owned bridge: the failure is handled by the bridges projector (degraded state per TC-FUNC-026). The bundles projector is not notified of the failure — no reverse dependency.
- Projector ordering: if projectors run in a defined order (e.g., bundles first, then automation, then bridges), the ordering is based on declared DependsOn edges, not on implicit "bundles creates records for other projectors" assumptions.
- Adding a new projector kind that a bundle can fan out to: the new projector must not introduce a DependsOn on bundles. This is a design invariant enforced by code review or compile-time check.
