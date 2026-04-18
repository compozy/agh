## TC-REG-012: Settings restart-aware save flow

**Priority:** P0
**Type:** Regression
**Status:** Not Run
**Estimated Time:** 15 minutes
**Created:** 2026-04-17
**Last Updated:** 2026-04-17
**Module:** Settings Save
**Route / Surface:** first shipped restart-aware Settings form, preferring General then Providers
**Design Reference:** `docs/design/paper/settings/AGH Settings — General@2x.png`, `docs/design/paper/settings/AGH Settings — Providers@2x.png`
**Execution Lane:** Manual + browser regression if preflight passes

### Objective

Verify at least one Settings form supports a persisted save flow that clearly communicates restart-required behavior and remains stable after navigation or reload.

### Preconditions

- [ ] `TC-REG-010` passed.
- [ ] A Settings section with restart-aware save semantics exists on the execution branch.
- [ ] Test data is safe to mutate and revert if needed.

### Test Steps

1. Open the chosen restart-aware Settings section.
   **Expected:** The form loads with editable values and clear save controls.
2. Change one safe setting value.
   **Expected:** The form becomes dirty and enables saving.
3. Save the change.
   **Expected:** The UI confirms the save and shows restart-required messaging or equivalent operator feedback.
4. Reload the route or navigate away and back.
   **Expected:** The saved value persists and the restart-required state remains coherent.
5. Revert the changed value if needed.
   **Expected:** Cleanup is possible without leaving the environment in a broken state.

### Edge Cases & Variations

| Variation | Input / State | Expected Result |
| --- | --- | --- |
| No restart indicator | save succeeds without restart copy | document whether the surface is not the correct restart-aware candidate and choose the next eligible section |
| Validation failure | invalid value entered | the form blocks save with field-level feedback |
| Save then reload | route refreshed immediately after save | persisted data and restart state remain consistent |

### Related Test Cases

- `TC-REG-010`
- `TC-REG-011`

### Notes

- Task_19 should prefer a form that clearly models restart-required behavior; if no such form exists despite Settings being present, that is a discrepancy worth documenting.
