## TC-FUNC-007: Network settings summary, restart-aware save, and operational link

**Priority:** P1
**Type:** Functional
**Status:** Not Run
**Estimated Time:** 12 minutes
**Created:** 2026-04-17
**Last Updated:** 2026-04-17
**Route:** `/settings/network`
**Traceability:** `task_11`, ADR-001, ADR-003, TechSpec > Runtime apply matrix

---

### Objective

Verify that the Network route shows runtime status and config together, keeps saves restart-aware, and links cleanly to the operational Network page.

---

### Preconditions

- [ ] HTTP is bound to loopback for the positive save path.
- [ ] Original field values are recorded before editing.

---

### Test Steps

1. Open `/settings/network`.
   - **Expected:** The route shows network config, runtime status summary, and a deep link to `/network`.

2. Edit one reversible network field.
   - **Expected:** The route becomes dirty and the save bar enables.

3. Save the change.
   - **Expected:** The result is presented as restart required, and the page remains usable after refetch.

4. Use the deep link to open `/network`.
   - **Expected:** The operational route opens successfully and reflects the expected app shell behavior.

5. Return to `/settings/network`.
   - **Expected:** The settings route remains stable and the nav highlights Network.

6. Restore the original network value.
   - **Expected:** Cleanup succeeds and leaves the environment stable for the dedicated non-loopback restriction case.

---

### Test Data

| Field | Value | Notes |
|-------|-------|-------|
| Route | `/settings/network` | Network settings route |
| Mutation target | Reversible network field | Prefer a low-risk field that can be restored immediately |

---

### Post-conditions

- Restore the edited network value.

---

### Edge Cases & Variations

| Variation | Input | Expected Result |
|-----------|-------|-----------------|
| Runtime status unavailable | Backend reports no live summary | Route still renders a graceful empty or unavailable state |
| Save with invalid value | Invalid host/port or other config | Validation error is shown without breaking the runtime summary |
| Return from operational page | Browser back from `/network` | Correct settings section remains active |

---

### Related Test Cases

- `TC-FUNC-006` covers the parallel Automation summary pattern.
- `TC-INT-013` covers the non-loopback negative path for mutation availability.

---

### Notes

- Do not treat non-loopback HTTP restriction behavior as a failure here; that is verified separately in `TC-INT-013`.
