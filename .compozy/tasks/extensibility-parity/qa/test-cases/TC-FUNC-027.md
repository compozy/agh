# TC-FUNC-027: Bundle activation rejects non-allowlisted owned kinds

**Priority:** P0
**Type:** Functional
**Package:** internal/bundles
**Related Tasks:** 12

## Objective

Validate that when a bundle.activation attempts to fan out and create owned resource records for a kind that is not in the bundle's declared allowlist, the activation is rejected before any records are written. This enforcement prevents bundles from creating arbitrary resource types and ensures each bundle explicitly declares which kinds it manages.

## Preconditions

- Resource runtime is active with all 10 resource kinds registered.
- A bundle resource "devtools-bundle" exists with an allowlist that permits: tool, hook.binding, skill. It does NOT include bridge.instance or automation.job.
- No bundle.activation records exist yet for this bundle.

## Test Steps

1. Create a bundle.activation for "devtools-bundle" that fans out to create owned records of kind tool and skill.
   **Expected:** Activation succeeds. Owned tool and skill resource records are created with correct owner_kind=bundle.activation and owner_id referencing this activation.

2. Verify the owned tool and skill records exist in the store with correct ownership metadata.
   **Expected:** Records are present. owner_kind and owner_id fields match the activation.

3. Attempt to create a bundle.activation for "devtools-bundle" that fans out to create an owned bridge.instance record.
   **Expected:** Activation is rejected with a validation error indicating that bridge.instance is not in the bundle's allowlist. No bridge.instance record is created. No partial writes occurred.

4. Verify no bridge.instance record was written to the store.
   **Expected:** Store query for bridge.instance records with owner matching this activation returns zero results.

5. Verify the previously created tool and skill records from step 1 are unaffected.
   **Expected:** Records are still present and unchanged.

6. Attempt to create a bundle.activation that mixes allowlisted (tool) and non-allowlisted (automation.job) kinds in a single fan-out.
   **Expected:** The entire activation is rejected atomically. Neither the tool nor the automation.job records are created. The validation fails on the non-allowlisted kind before any writes occur.

7. Verify no records were written from the mixed activation.
   **Expected:** No new tool or automation.job records exist with the rejected activation's owner reference.

## Edge Cases

- Bundle with an empty allowlist: any activation that attempts to fan out to any kind is rejected.
- Bundle allowlist containing a kind that does not exist in the resource runtime (e.g., typo "toool"): the bundle definition itself should fail validation at creation time, not at activation time.
- Activation with zero owned records (empty fan-out): succeeds — an activation with no dependencies is valid.
- Activation referencing a bundle that has been deleted: activation is rejected (bundle must exist at activation time).
- Updating a bundle's allowlist to remove a kind that existing activations own: verify whether existing owned records are orphaned or whether the update is blocked until activations are removed.
