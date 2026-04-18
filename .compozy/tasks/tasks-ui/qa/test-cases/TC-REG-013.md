## TC-REG-013: Settings collection CRUD flow

**Priority:** P1
**Type:** Regression
**Status:** Not Run
**Estimated Time:** 18 minutes
**Created:** 2026-04-17
**Last Updated:** 2026-04-17
**Module:** Settings Collection CRUD
**Route / Surface:** first shipped collection editor, preferring Environments then Providers
**Design Reference:** `docs/design/paper/settings/AGH Settings — Environments@2x.png`, `docs/design/paper/settings/AGH Settings — Providers@2x.png`
**Execution Lane:** Manual + targeted browser regression if preflight passes

### Objective

Verify one Settings collection surface supports create, edit, and delete lifecycle operations with stable list refresh and persistence semantics.

### Preconditions

- [ ] `TC-REG-010` passed.
- [ ] A collection-style Settings surface exists on the execution branch.
- [ ] Test data can be added and removed safely.

### Test Steps

1. Open the chosen collection Settings surface.
   **Expected:** A list or table of existing items is visible with create/edit affordances.
2. Create a new item with valid data.
   **Expected:** The new item appears in the collection without requiring a hard refresh.
3. Edit the newly created item.
   **Expected:** The updated values persist and the visible list reflects the change.
4. Delete or remove the same item.
   **Expected:** The collection returns to its original state and no orphaned row remains.

### Edge Cases & Variations

| Variation | Input / State | Expected Result |
| --- | --- | --- |
| Empty collection | no items exist initially | create flow still works and the empty state recovers into a list state |
| Duplicate value | create or edit uses an existing identifier | the UI surfaces a validation or conflict error cleanly |
| Reload after mutation | route refreshed after create or delete | the collection state stays consistent with the last successful mutation |

### Related Test Cases

- `TC-REG-011`
- `TC-REG-012`
- `TC-REG-014`

### Notes

- This case stays P1 because the restart-aware save and shell checks are the higher-priority Settings regressions.
