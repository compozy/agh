# TC-FUNC-028: Owner-indexed cleanup deletes only owned records

**Priority:** P1
**Type:** Functional
**Package:** internal/bundles
**Related Tasks:** 12

## Objective

Validate that when a bundle.activation is deleted, the cleanup process removes only resource records whose owner_kind and owner_id match the deleted activation. Records owned by other activations, records with no owner, and records owned by different owner kinds must remain completely untouched. This tests the correctness of the owner-indexed deletion path.

## Preconditions

- Resource runtime is active with bundles support.
- Bundle "devtools-bundle" exists with allowlist: tool, skill, hook.binding.
- Two bundle.activations exist:
  - Activation A: owns tool "lint-tool" and skill "lint-skill".
  - Activation B: owns tool "format-tool" and hook.binding "format-hook".
- Additionally, a standalone tool "user-tool" exists with no owner (created directly by a user).
- A hook.binding "session-hook" exists owned by a different owner kind (e.g., owner_kind=extension).

## Test Steps

1. Verify initial state: query all tool, skill, and hook.binding records.
   **Expected:** Five records exist: "lint-tool" (owned by A), "lint-skill" (owned by A), "format-tool" (owned by B), "format-hook" (owned by B), "user-tool" (no owner), "session-hook" (owned by extension).

2. Delete Activation A.
   **Expected:** Deletion succeeds without error.

3. Query all tool records.
   **Expected:** "lint-tool" is gone. "format-tool" and "user-tool" still exist.

4. Query all skill records.
   **Expected:** "lint-skill" is gone. No other skill records are affected.

5. Query all hook.binding records.
   **Expected:** "format-hook" and "session-hook" still exist. Neither was touched by the deletion of Activation A.

6. Verify Activation B is still active and its owned records are intact.
   **Expected:** Activation B exists. "format-tool" and "format-hook" have correct ownership pointing to B.

7. Verify "user-tool" has no owner and was not modified.
   **Expected:** "user-tool" exists with no owner_kind and no owner_id. Version unchanged.

8. Verify "session-hook" retains its original owner (extension).
   **Expected:** "session-hook" exists with owner_kind=extension. Version unchanged.

## Edge Cases

- Deleting an activation that owns zero records: deletion succeeds with no side effects on the store.
- Deleting an activation whose owned records were already individually deleted: no error — the cleanup is idempotent.
- Two activations owning records of the same kind with similar names (e.g., "tool-a" owned by A, "tool-a-v2" owned by B): only the correct one is deleted based on owner_id, not name pattern matching.
- Deleting both activations in rapid succession: no race conditions. Each cleanup correctly targets only its own records.
- Re-creating an activation with the same name after deletion: new activation gets a new owner_id. It does not inherit or resurrect the deleted activation's owned records.
