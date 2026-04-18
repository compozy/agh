## TC-FUNC-002: General settings restart-required save and daemon restart flow

**Priority:** P0
**Type:** Functional
**Status:** Not Run
**Estimated Time:** 20 minutes
**Created:** 2026-04-17
**Last Updated:** 2026-04-17
**Route:** `/settings/general`
**Traceability:** `task_10`, ADR-003, TechSpec > Runtime apply matrix, Settings actions

---

### Objective

Verify that the General page separates runtime and config state, saves restart-required configuration correctly, and drives the daemon restart action with durable status polling that survives refresh.

---

### Preconditions

- [ ] AGH is running in detached daemon mode from a local build.
- [ ] HTTP is bound to loopback so settings mutations and restart are allowed.
- [ ] The executor has permission to restart the local daemon safely.
- [ ] Record the original field values before editing so they can be restored after the test.

---

### Test Steps

1. Open `/settings/general`.
   - **Expected:** The page shows runtime cards, config-path information, editable defaults/limits/session fields, and a visible restart action.

2. Record the current values for one reversible field, then edit one restart-required field such as session timeout or a default selection.
   - **Expected:** The page becomes dirty, the save bar enables, and no runtime card changes immediately.

3. Save the change.
   - **Expected:** The save succeeds, the page shows a restart-required result, any warnings are visible, and the restart banner/action state becomes relevant for the route.

4. Trigger `Restart daemon`.
   - **Expected:** The action enters a pending/polling state, the restart control disables while in progress, and the UI exposes a durable operation state instead of assuming synchronous success.

5. Refresh the page while restart polling is still in progress.
   - **Expected:** The page reloads, restores the restart status, and continues polling instead of forgetting the in-flight operation.

6. Wait for a terminal restart result.
   - **Expected:** The banner resolves to `ready` or `failed` with explicit status messaging; if failed, a failure reason is visible or discoverable.

7. Restore the original value using the same route once the daemon is stable again.
   - **Expected:** Cleanup saves behave like the original mutation and leave the environment ready for later cases.

---

### Test Data

| Field | Value | Notes |
|-------|-------|-------|
| Route | `/settings/general` | General settings route |
| Mutation target | Reversible restart-required field | Prefer a low-risk field such as session timeout |
| Evidence | `operation_id`, terminal status | Record in verification report |

---

### Post-conditions

- Restore any edited General config values to their original state.
- Confirm the daemon is healthy before moving to later tests.
- Capture at least one screenshot of the restart banner state.

---

### Edge Cases & Variations

| Variation | Input | Expected Result |
|-----------|-------|-----------------|
| Active sessions exist during restart | Non-zero `active_session_count` | The operator sees a warning or clear count before or during restart |
| Restart helper failure | Trigger restart in a failing environment | Terminal status becomes `failed` with visible diagnostics |
| Restart action unavailable | Action metadata reports unavailable | The control is disabled and the reason is not misrepresented as a save failure |

---

### Related Test Cases

- `TC-FUNC-001` proves the shell and route entrypoints before this case runs.
- `TC-FUNC-003` validates an action-trigger route that must stay separate from restart messaging.
- `TC-INT-013` validates the negative-path mutation restriction behavior on non-loopback HTTP.

---

### Notes

- This is the primary restart P0 case for the entire feature and must be reflected in screenshots plus the verification report.
