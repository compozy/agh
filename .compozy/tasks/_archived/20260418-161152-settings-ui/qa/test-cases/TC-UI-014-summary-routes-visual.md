## TC-UI-014: Settings shell and summary routes visual validation against Paper exports

**Priority:** P1
**Type:** UI
**Status:** Not Run
**Estimated Time:** 25 minutes
**Created:** 2026-04-17
**Last Updated:** 2026-04-17
**Route:** `/settings`, `/settings/general`, `/settings/memory`, `/settings/observability`, `/settings/skills`, `/settings/automation`, `/settings/network`
**Traceability:** TechSpec > Design References, `task_10`, `task_11`

---

### Objective

Verify that the settings shell and the six summary/diagnostic routes match the local Paper exports closely enough in layout, hierarchy, spacing, and breakpoint behavior for release execution.

---

### Preconditions

- [ ] The executor can open the local Paper PNG exports under `docs/design/paper/settings/`.
- [ ] Browser viewports for `1280`, `768`, and `375` are available.
- [ ] The settings shell and summary routes load without blocking functional errors.

---

### Test Steps

1. Compare `/settings` and the left navigation shell against the live app at desktop width.
   - **Expected:** The nav width, section ordering, overall composition, and shell framing match the intended design direction from the Paper references.

2. Compare `/settings/general`, `/settings/memory`, and `/settings/observability` against their matching Paper exports.
   - **Expected:** Card grouping, heading hierarchy, status-line placement, and save-bar placement follow the corresponding artboards.

3. Compare `/settings/skills`, `/settings/automation`, and `/settings/network` against their matching Paper exports.
   - **Expected:** Summary blocks, deep-link placement, and restart banner/save-bar composition align with the exported references.

4. Resize the viewport to `768`.
   - **Expected:** The shell and summary routes remain readable, cards stack or wrap cleanly, and no primary content is clipped.

5. Resize the viewport to `375`.
   - **Expected:** The layout remains legible, controls stay reachable, and the save bar or banners do not obscure essential content.

---

### Test Data

| Field | Value | Notes |
|-------|-------|-------|
| Desktop viewport | `1280px` | Primary comparison view |
| Tablet viewport | `768px` | Responsive check |
| Mobile viewport | `375px` | Responsive check |
| Design sources | Local `2880x1800` PNG exports | Figma is not available for this task |

---

### Post-conditions

- Capture screenshots for any route with meaningful visual drift.
- File a bug if hierarchy, spacing, or responsive behavior materially diverges from the Paper references.

---

### Edge Cases & Variations

| Variation | Input | Expected Result |
|-----------|-------|-----------------|
| Restart banner visible | Use a route with pending restart state | Banner placement stays aligned and does not occlude content |
| Save bar active | Dirty form state present | Bottom action region stays usable at all viewports |
| Empty or zero-state summary | Low-runtime activity | Visual hierarchy still holds without collapsing awkwardly |

---

### Related Test Cases

- `TC-UI-015` validates the collection and hybrid routes visually.
- `TC-FUNC-001` validates shell behavior functionally.

---

### Notes

- Use screenshots named with the case ID and route slug to keep visual evidence organized for `task_16`.
