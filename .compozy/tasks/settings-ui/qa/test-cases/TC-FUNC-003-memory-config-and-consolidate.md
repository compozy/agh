## TC-FUNC-003: Memory settings restart-aware save and consolidate action

**Priority:** P1
**Type:** Functional
**Status:** Not Run
**Estimated Time:** 15 minutes
**Created:** 2026-04-17
**Last Updated:** 2026-04-17
**Route:** `/settings/memory`
**Traceability:** `task_10`, ADR-003, TechSpec > Runtime apply matrix, Settings actions

---

### Objective

Verify that the Memory page shows config and health state together, treats config saves as restart-required, and keeps the `Trigger now` consolidate action separate from save-state messaging.

---

### Preconditions

- [ ] AGH is running with memory features enabled.
- [ ] HTTP is bound to loopback so mutations are allowed.
- [ ] The executor records original field values before editing.

---

### Test Steps

1. Open `/settings/memory`.
   - **Expected:** The page shows memory configuration fields, dream/consolidation settings, health/runtime information, and a visible manual consolidate action if the backend exposes it.

2. Edit one reversible config-backed memory field.
   - **Expected:** The page becomes dirty and the save bar enables.

3. Save the config change.
   - **Expected:** The mutation succeeds with restart-required messaging and does not present itself as an immediate-apply change.

4. Trigger the manual consolidate action.
   - **Expected:** The page shows a distinct pending/result state for the action, and the action does not publish a restart-required banner by itself.

5. After the action completes, verify the route still shows the prior save result and the page remains usable.
   - **Expected:** Save feedback and action feedback remain distinct, and no stale pending state leaks across them.

6. Restore the original config value.
   - **Expected:** Cleanup succeeds and returns the route to baseline.

---

### Test Data

| Field | Value | Notes |
|-------|-------|-------|
| Route | `/settings/memory` | Memory settings route |
| Mutation target | Reversible memory config field | Pick a field that can be restored immediately |
| Action | Consolidate / Trigger now | Uses the existing memory action path |

---

### Post-conditions

- Restore edited fields to their original values.
- Note any discrepancy between action feedback and restart feedback as a bug candidate.

---

### Edge Cases & Variations

| Variation | Input | Expected Result |
|-----------|-------|-----------------|
| Consolidate unavailable | Action metadata reports unavailable | Button/action control is disabled or hidden with clear explanation |
| Repeated clicks while action is pending | Trigger action twice quickly | Duplicate execution is prevented or clearly serialized |
| Empty / low-activity memory state | No recent consolidate history | The route still renders a valid health summary |

---

### Related Test Cases

- `TC-FUNC-002` validates the cross-route restart flow this case depends on for cleanup.
- `TC-FUNC-004` validates another diagnostics-heavy page with restart-required saves.

---

### Notes

- This case is important because it mixes one restart-required config flow with one manual action flow on the same route.
