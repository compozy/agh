## TC-FUNC-008: Providers collection CRUD and builtin fallback behavior

**Priority:** P0
**Type:** Functional
**Status:** Not Run
**Estimated Time:** 18 minutes
**Created:** 2026-04-17
**Last Updated:** 2026-04-17
**Route:** `/settings/providers`
**Traceability:** `task_12`, ADR-002, TechSpec > Collection mutation semantics

---

### Objective

Verify that the Providers route supports list/detail/edit/delete workflows, surfaces source metadata clearly, and reveals builtin fallback behavior when an overlay definition is deleted.

---

### Preconditions

- [ ] HTTP is bound to loopback so collection mutations are allowed.
- [ ] At least one builtin provider exists in the environment.
- [ ] The executor can create or edit an overlay definition for a provider name that also has a builtin definition.
- [ ] Original provider data is recorded before editing.

---

### Test Steps

1. Open `/settings/providers`.
   - **Expected:** The route shows the provider list, detail panel, source metadata, and create/edit/delete controls.

2. Select a provider that has a builtin definition and create or edit an overlay-backed definition for it.
   - **Expected:** The editor reflects full-replacement semantics and makes the selected provider and source metadata explicit.

3. Save the overlay definition.
   - **Expected:** The save succeeds, the detail view refreshes with the edited values, and the route reports a restart-aware collection mutation result.

4. Re-open the same provider detail.
   - **Expected:** The detail view shows the overlay-backed values and identifies the current source correctly.

5. Delete the overlay-backed definition.
   - **Expected:** The route confirms the delete, refetches the collection, and the builtin provider definition becomes visible again instead of disappearing completely.

6. Verify the fallback detail after delete.
   - **Expected:** The provider still exists in the list, but its detail/source metadata now reflect the builtin definition.

---

### Test Data

| Field | Value | Notes |
|-------|-------|-------|
| Route | `/settings/providers` | Providers collection route |
| Test provider | Builtin provider with temporary overlay | Required for fallback coverage |
| Optional temp name | `qa-provider-temp` | Use only if an additional disposable provider is needed |

---

### Post-conditions

- Remove any temporary overlay-only provider created for the test.
- Restore modified provider fields to their original values if the builtin-fallback flow was not used for cleanup.
- Capture a screenshot if source metadata or fallback messaging is unclear.

---

### Edge Cases & Variations

| Variation | Input | Expected Result |
|-----------|-------|-----------------|
| Builtin-only provider delete attempt | Provider has no overlay | Delete is blocked or clearly explained |
| Duplicate provider name | Save with existing conflicting name | Inline conflict/validation error is shown |
| Overlay delete with no builtin underneath | Temporary provider only | Entry disappears cleanly after delete |

---

### Related Test Cases

- `TC-FUNC-009` validates the parallel Environments collection route.
- `TC-FUNC-010` validates the more complex collection semantics on MCP Servers.

---

### Notes

- This is the primary P0 collection CRUD case for the feature because it proves both edit and fallback semantics on a single route.
