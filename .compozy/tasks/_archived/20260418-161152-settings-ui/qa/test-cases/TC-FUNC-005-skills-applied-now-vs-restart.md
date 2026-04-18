## TC-FUNC-005: Skills settings applied-now mutation versus restart-required policy save

**Priority:** P0
**Type:** Functional
**Status:** Not Run
**Estimated Time:** 18 minutes
**Created:** 2026-04-17
**Last Updated:** 2026-04-17
**Route:** `/settings/skills`
**Traceability:** `task_11`, ADR-003, TechSpec > Runtime apply matrix

---

### Objective

Verify that the Skills route clearly separates immediate disabled-skill changes from restart-required marketplace or policy edits, and preserves the deep link into the operational Skills surface.

---

### Preconditions

- [ ] HTTP is bound to loopback so mutations are allowed.
- [ ] At least one skill entry is available to disable or re-enable.
- [ ] Original values are recorded before editing.

---

### Test Steps

1. Open `/settings/skills`.
   - **Expected:** The route shows disabled-skill state, marketplace/policy state, runtime/discovery summaries, and a deep link to `/skills`.

2. Change the disabled-skill state using the immediate-apply control path on the route.
   - **Expected:** Only the relevant card becomes dirty or actionable; the entire page does not collapse into one ambiguous save flow.

3. Save the disabled-skill change.
   - **Expected:** The result is presented as applied now, no restart-required banner appears for this action, and the visible counts/state update after refetch.

4. Edit one marketplace or policy field that is restart-required.
   - **Expected:** The second edit is isolated to its own save path and does not reuse the applied-now success state from the previous step.

5. Save the marketplace or policy change.
   - **Expected:** The result is presented as restart required, with restart-specific messaging distinct from the prior applied-now change.

6. Use the deep link to open `/skills`, then navigate back to `/settings/skills`.
   - **Expected:** The operational page opens correctly and returning to the settings route preserves the expected section context.

7. Restore the original skills settings.
   - **Expected:** Cleanup succeeds without leaving the route in a mixed pending state.

---

### Test Data

| Field | Value | Notes |
|-------|-------|-------|
| Route | `/settings/skills` | Skills settings route |
| Immediate-change sample | Disabled skill toggle/list update | Expected `applied_now` |
| Restart-change sample | Marketplace or policy field | Expected `restart_required` |

---

### Post-conditions

- Restore edited skills values.
- Record screenshots if applied-now and restart-required messaging collide visually.

---

### Edge Cases & Variations

| Variation | Input | Expected Result |
|-----------|-------|-----------------|
| Mixed-behavior change attempt | Edit both immediate and restart-required fields before saving | UI keeps them split into separate save paths or otherwise avoids one ambiguous mutation |
| No discovered skills | Empty discovery state | Disabled-skill controls degrade gracefully |
| Deep link return | Browser back from `/skills` | Returns to the expected settings section |

---

### Related Test Cases

- `TC-FUNC-006` and `TC-FUNC-007` validate the other summary routes with deep links.
- `TC-FUNC-012` validates another route with mixed mutation semantics, but around hooks and extensions.

---

### Notes

- This case is the main P0 proof that `applied_now` behavior exists in the settings surface and is not lost behind restart messaging.
