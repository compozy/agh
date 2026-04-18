## TC-FUNC-012: Hooks and Extensions hybrid config and immediate-action behavior

**Priority:** P0
**Type:** Functional
**Status:** Not Run
**Estimated Time:** 20 minutes
**Created:** 2026-04-17
**Last Updated:** 2026-04-17
**Route:** `/settings/hooks-extensions`
**Traceability:** `task_14`, ADR-001, ADR-003, ADR-004, TechSpec > Data Models, Runtime apply matrix

---

### Objective

Verify that the Hooks & Extensions route keeps restart-aware hook or policy edits separate from immediate extension actions, while preserving clear runtime/config boundaries on one page.

---

### Preconditions

- [ ] HTTP is bound to loopback for the positive mutation path.
- [ ] At least one installed extension is present, or the route exposes a deterministic installed-extension fixture.
- [ ] At least one hook declaration is visible on the route.
- [ ] Original hook/policy state is recorded before editing.

---

### Test Steps

1. Open `/settings/hooks-extensions`.
   - **Expected:** The route shows separate areas for hook declarations, installed extension runtime state, and extension marketplace/resource policy config.

2. Change one hook declaration or hook-enabled state.
   - **Expected:** The change is treated as config-backed and prepares a restart-aware save path.

3. Save the hook/config change.
   - **Expected:** The result is shown as restart required and is clearly separate from extension runtime actions.

4. Toggle one installed extension enable/disable action.
   - **Expected:** The extension action runs immediately, shows its own progress/result state, and does not publish a restart-required banner.

5. Edit one extension marketplace or resource policy field.
   - **Expected:** The route exposes a second restart-required config save path distinct from the extension toggle result.

6. Save the policy change.
   - **Expected:** The route reports restart-required behavior for the policy change while keeping the installed-extension runtime list visible.

7. Restore the original hook and policy state.
   - **Expected:** Cleanup succeeds and extension runtime state remains coherent.

---

### Test Data

| Field | Value | Notes |
|-------|-------|-------|
| Route | `/settings/hooks-extensions` | Hybrid settings route |
| Extension sample | Any installed extension | Used for immediate enable/disable |
| Hook sample | Any visible hook declaration | Used for restart-aware config save |

---

### Post-conditions

- Restore original hook and policy settings.
- Return the extension to its original enabled/disabled state if it was toggled.
- Capture a screenshot if the route conflates immediate actions with restart-required saves.

---

### Edge Cases & Variations

| Variation | Input | Expected Result |
|-----------|-------|-----------------|
| Mutation unavailable | Transport/policy blocks extension action | Controls are disabled or clearly explained without fake success |
| Hook toggle save after extension action | Run immediate action first, then hook save | Previous action status does not replace the restart-required save result |
| Policy save after hook save | Two restart-aware saves in sequence | Latest save result replaces the earlier one without affecting runtime extension state |

---

### Related Test Cases

- `TC-FUNC-005` validates another route with mixed mutation semantics.
- `TC-INT-013` validates the non-loopback restriction variant for this route.

---

### Notes

- This is the primary P0 proof that the product distinguishes config-backed settings from operational extension actions on the same screen.
