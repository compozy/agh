## TC-FUNC-001: Settings shell navigation and section entrypoints

**Priority:** P0
**Type:** Functional
**Status:** Not Run
**Estimated Time:** 10 minutes
**Created:** 2026-04-17
**Last Updated:** 2026-04-17
**Route:** `/settings`
**Traceability:** `task_08`, `task_09`, TechSpec > Web route coverage

---

### Objective

Verify that the shared `/settings` shell loads, exposes all 10 settings sections in the expected order, and supports direct entry plus route-to-route navigation without breaking the shell frame.

---

### Preconditions

- [ ] AGH daemon and web UI are running from the current branch.
- [ ] The executor can open the web app over HTTP.
- [ ] No blocking browser console/runtime errors are present before starting.

---

### Test Steps

1. Open `/settings`.
   - **Expected:** The settings shell loads with the left navigation rail visible and the index placeholder rendered in the content pane.

2. Verify the section order in the left navigation.
   - **Expected:** The nav lists `General`, `Providers`, `MCP Servers`, `Environments`, `Memory`, `Skills`, `Automation`, `Network`, `Observability`, and `Hooks & Extensions` in that exact order.

3. Click `General`, then `Providers`, then `Hooks & Extensions`.
   - **Expected:** The shell frame stays mounted, the active nav indicator moves with the selected section, and each route loads its own page content.

4. Enter `/settings/network` directly in the browser address bar.
   - **Expected:** The browser deep-links to the Network page, the shell still renders, and the Network nav item is marked active.

5. Use browser refresh on `/settings/network`.
   - **Expected:** The page reloads into the same section without redirecting away from settings or losing the shared shell.

6. Use browser back/forward between two settings routes.
   - **Expected:** The active nav item and displayed route content stay in sync with browser history.

---

### Test Data

| Field | Value | Notes |
|-------|-------|-------|
| Base route | `/settings` | Shared shell entrypoint |
| Deep-link route | `/settings/network` | Direct-entry validation |

---

### Post-conditions

- No cleanup required.
- Record a screenshot of the shell if a visual/navigation defect appears.

---

### Edge Cases & Variations

| Variation | Input | Expected Result |
|-----------|-------|-----------------|
| Direct entry to a child route | `/settings/hooks-extensions` | Shell still renders and the matching nav item is active |
| Browser refresh on child route | Refresh on `/settings/providers` | Same route reloads without a blank shell |
| Rapid section switching | Click multiple nav items quickly | No shell crash, stale overlay, or broken active state |

---

### Related Test Cases

- `TC-FUNC-002` validates a restart-required summary route inside the shell.
- `TC-FUNC-008` validates collection behavior after shell navigation succeeds.
- `TC-UI-014` validates the shell against the Paper references.

---

### Notes

- This case is the first smoke gate because every other settings route depends on the shell remaining stable.
