## TC-FUNC-006: Automation settings summary, restart-aware save, and operational link

**Priority:** P1
**Type:** Functional
**Status:** Not Run
**Estimated Time:** 12 minutes
**Created:** 2026-04-17
**Last Updated:** 2026-04-17
**Route:** `/settings/automation`
**Traceability:** `task_11`, ADR-001, ADR-003, TechSpec > Web route coverage

---

### Objective

Verify that the Automation route presents configuration and runtime summary data, treats saves as restart-required, and links cleanly into the operational Automation page.

---

### Preconditions

- [ ] HTTP is bound to loopback so settings mutations are allowed.
- [ ] The executor records any edited values before saving.

---

### Test Steps

1. Open `/settings/automation`.
   - **Expected:** The route shows automation configuration, runtime summary counts, and a deep link to `/automation`.

2. Edit one reversible automation field.
   - **Expected:** The route becomes dirty and the save bar enables.

3. Save the change.
   - **Expected:** The mutation result is shown as restart required and the summary state remains visible after refetch.

4. Use the deep link to open `/automation`.
   - **Expected:** The operational Automation route loads without breaking the app shell.

5. Return to `/settings/automation`.
   - **Expected:** The settings route is reachable again and shows the expected active section.

6. Restore the original automation value.
   - **Expected:** Cleanup succeeds and leaves the route stable.

---

### Test Data

| Field | Value | Notes |
|-------|-------|-------|
| Route | `/settings/automation` | Automation settings route |
| Mutation target | Reversible automation field | Choose a field that is safe to restore |

---

### Post-conditions

- Restore the edited automation value.

---

### Edge Cases & Variations

| Variation | Input | Expected Result |
|-----------|-------|-----------------|
| Empty runtime summary | Zero jobs or triggers active | Runtime cards still render valid counts |
| Save failure | Invalid payload or backend rejection | Error stays localized to the route and does not hide summary information |
| Link navigation | Open `/automation` in same tab | Return path to settings remains obvious |

---

### Related Test Cases

- `TC-FUNC-005` validates the deeper mixed-behavior summary route.
- `TC-FUNC-007` validates the parallel Network summary route.

---

### Notes

- This route is intentionally configuration-oriented; do not expand this case into full operational Automation console testing.
