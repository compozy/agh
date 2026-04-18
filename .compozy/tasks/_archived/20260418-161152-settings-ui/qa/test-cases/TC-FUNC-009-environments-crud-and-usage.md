## TC-FUNC-009: Environments collection CRUD and usage-count behavior

**Priority:** P1
**Type:** Functional
**Status:** Not Run
**Estimated Time:** 15 minutes
**Created:** 2026-04-17
**Last Updated:** 2026-04-17
**Route:** `/settings/environments`
**Traceability:** `task_12`, ADR-002, TechSpec > Data Models, Collection mutation semantics

---

### Objective

Verify that the Environments route supports list/detail/create/edit/delete flows while exposing usage counts, replace semantics, and validation or conflict errors clearly.

---

### Preconditions

- [ ] HTTP is bound to loopback so collection mutations are allowed.
- [ ] A disposable environment name is available, for example `qa-env-temp`.
- [ ] The executor records any original data before editing an existing environment.

---

### Test Steps

1. Open `/settings/environments`.
   - **Expected:** The route shows the environment list, detail state, usage counts, and create/edit/delete controls.

2. Create a new environment such as `qa-env-temp`.
   - **Expected:** The editor supports full-replacement input for the new environment and keeps list selection stable.

3. Save the new environment.
   - **Expected:** The collection refetches, the new environment appears in the list, and the detail panel shows the saved values plus source metadata.

4. Edit the same environment and save again.
   - **Expected:** The update behaves as a full replacement rather than a partial hidden merge, and the detail panel updates after refetch.

5. Verify the usage-count display for at least one environment in the list.
   - **Expected:** The route shows workspace usage counts or equivalent summary data without breaking CRUD state.

6. Delete `qa-env-temp`.
   - **Expected:** The route removes the environment cleanly and returns the detail panel to the expected post-delete state.

---

### Test Data

| Field | Value | Notes |
|-------|-------|-------|
| Route | `/settings/environments` | Environments collection route |
| Temporary environment | `qa-env-temp` | Disposable test record |

---

### Post-conditions

- Ensure `qa-env-temp` is deleted.
- Restore any edited non-temporary environment to baseline.

---

### Edge Cases & Variations

| Variation | Input | Expected Result |
|-----------|-------|-----------------|
| Duplicate name | Create or rename to an existing environment | Validation/conflict error is shown inline |
| In-use environment delete | Environment with non-zero usage count | Product either prevents delete or explains consequences clearly |
| Empty-state collection | No custom environments exist | Empty state renders without broken detail UI |

---

### Related Test Cases

- `TC-FUNC-008` covers the other collection page with fallback semantics.
- `TC-UI-015` covers the visual structure of the collection routes.

---

### Notes

- Prefer a temporary environment so cleanup is deterministic.
