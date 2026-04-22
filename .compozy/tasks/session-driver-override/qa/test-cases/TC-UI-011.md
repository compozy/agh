## TC-UI-011: Resume Failure Renders a Dedicated Inline State With Session ID and Missing Provider

**Priority:** P0
**Type:** UI
**Status:** Not Run
**Estimated Time:** 15 minutes
**Created:** 2026-04-21
**Last Updated:** 2026-04-21
**Module:** Web resume-failure UX
**Traceability:** Task 06 resume-failure requirements; ADR-003 and ADR-004; TechSpec UI integration tests and "Known Risks"

---

### Objective

Verify that when a persisted provider becomes unavailable, the web session route renders a dedicated inline failure state that explains the issue using the session id and missing provider from the backend.

---

### Preconditions

- [ ] The backend scenario from `TC-INT-005` is prepared with a persisted session whose provider is no longer available.
- [ ] Browser execution environment is available.
- [ ] Network and screenshot capture are available.

---

### Test Steps

1. Navigate to the affected session route or attempt to resume the affected session from the web app.
   **Expected:** The backend returns the explicit unavailable-provider failure for that session.

2. Observe the session page after the failed resume.
   **Expected:** The page renders a dedicated inline failure state rather than relying only on a toast or hidden console output.

3. Inspect the failure copy.
   **Expected:** The UI includes the session id and missing provider, matching the backend error payload closely enough for the operator to act.

4. Refresh or revisit the route.
   **Expected:** The inline failure state remains reproducible and actionable until the underlying provider/config issue is fixed.

---

### Evidence to Capture

- Screenshot of the inline failure panel.
- Network capture of the failed resume response showing session id and provider.
- Optional log snippet or console capture showing the client handled the explicit failure path without falling back silently.

---

### Edge Cases & Variations

| Variation | Input | Expected Result |
| --- | --- | --- |
| First navigation to the route | Missing persisted provider | Inline failure renders immediately. |
| Refresh on the same route | Same missing provider | Failure remains stable and actionable. |
| Toast also appears | Same failure | Inline panel is still present; toast-only behavior is not sufficient. |

---

### Related Test Cases

- `TC-INT-005` for the backend failure semantics
- `TC-UI-010` for the successful create flow baseline

---

### Notes

Task 08 should store the screenshot in `qa/screenshots/` and cross-link it from the final verification report.
