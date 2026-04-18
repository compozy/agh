## TC-FUNC-004: Observability diagnostics and log-tail capability metadata

**Priority:** P1
**Type:** Functional
**Status:** Not Run
**Estimated Time:** 12 minutes
**Created:** 2026-04-17
**Last Updated:** 2026-04-17
**Route:** `/settings/observability`
**Traceability:** `task_10`, TechSpec > Data Models, API Endpoints, Runtime apply matrix

---

### Objective

Verify that the Observability page exposes config-backed policy, runtime diagnostics, and log-tail capability metadata without confusing read-only diagnostics with editable settings.

---

### Preconditions

- [ ] AGH is running with observable log and database state.
- [ ] HTTP is bound to loopback so a save path can be exercised.
- [ ] The executor can inspect log-tail metadata without needing to change unrelated runtime behavior.

---

### Test Steps

1. Open `/settings/observability`.
   - **Expected:** The route shows observability/transcript config, database or storage metrics, and a log-tail capability or stream entrypoint.

2. Record the log-tail metadata shown by the page.
   - **Expected:** The route exposes a recognizable stream capability such as `/api/settings/observability/log-tail` or equivalent UI metadata.

3. Edit one reversible config-backed field.
   - **Expected:** The page becomes dirty and the save bar enables.

4. Save the change.
   - **Expected:** The mutation succeeds with restart-required messaging, and runtime diagnostics remain visible after refetch.

5. Restore the original field value.
   - **Expected:** Cleanup succeeds and returns the route to baseline.

---

### Test Data

| Field | Value | Notes |
|-------|-------|-------|
| Route | `/settings/observability` | Observability settings route |
| Log-tail path | `/api/settings/observability/log-tail` | Expected capability path from the TechSpec |
| Mutation target | Reversible observability field | Choose a field that is easy to restore |

---

### Post-conditions

- Restore edited values to baseline.
- Capture a screenshot if the route fails to expose runtime diagnostics or log-tail metadata.

---

### Edge Cases & Variations

| Variation | Input | Expected Result |
|-----------|-------|-----------------|
| Zero metrics state | Fresh or empty runtime metrics | The route still renders a valid zero-state instead of crashing |
| Log-tail temporarily unavailable | Capability disabled or backend unavailable | The UI explains the limitation instead of showing a broken control |
| Save with no changes | No field edits | Save stays disabled or no mutation is sent |

---

### Related Test Cases

- `TC-FUNC-003` validates another diagnostics-heavy page with a manual action.
- `TC-UI-014` verifies this route visually against its Paper export.

---

### Notes

- This case does not require deep log streaming validation; it verifies that the route exposes the expected operator entrypoint and surrounding state.
