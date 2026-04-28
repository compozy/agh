## TC-UI-015: Collection and hybrid settings routes visual validation against Paper exports

**Priority:** P1
**Type:** UI
**Status:** Not Run
**Estimated Time:** 25 minutes
**Created:** 2026-04-17
**Last Updated:** 2026-04-17
**Route:** `/settings/providers`, `/sandbox`, `/settings/mcp-servers`, `/settings/hooks-extensions`
**Traceability:** TechSpec > Design References, `task_12`, `task_13`, `task_14`

---

### Objective

Verify that the collection-oriented and hybrid settings routes match the local Paper exports in list/detail composition, dialog presentation, precedence badges, scope controls, and hybrid layout hierarchy.

---

### Preconditions

- [ ] The executor can open the local Paper PNG exports under `docs/design/paper/settings/`.
- [ ] Browser viewports for `1280`, `768`, and `375` are available.
- [ ] At least one list item exists for each collection route, or deterministic seed data is available.

---

### Test Steps

1. Compare `/settings/providers` and `/sandbox` with their Paper exports.
   - **Expected:** List/detail layout, section framing, metadata placement, and dialog affordances align with the design references.

2. Open one create/edit dialog on a collection page.
   - **Expected:** The dialog spacing, labels, buttons, and form hierarchy remain visually consistent with the route design and do not feel detached from the page shell.

3. Compare `/settings/mcp-servers` with its Paper export.
   - **Expected:** Scope chips, precedence badges, target controls, and explanatory copy are visually legible and consistent with the reference artboard.

4. Compare `/settings/hooks-extensions` with its Paper export.
   - **Expected:** Hook config, extension runtime cards, and policy sections remain visually distinct while still reading as one coherent page.

5. Resize to `768` and `375`.
   - **Expected:** Collection lists, dialogs, scope chips, and hybrid panels remain usable without clipped controls or unreadable metadata.

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

- Capture screenshots for any material visual mismatch.
- File a bug if list/detail rhythm, dialog layout, scope controls, or hybrid grouping diverge materially from the Paper exports.

---

### Edge Cases & Variations

| Variation | Input | Expected Result |
|-----------|-------|-----------------|
| Empty collection state | No custom records | Empty-state layout remains intentional and readable |
| Workspace scope active | `/settings/mcp-servers` in workspace scope | Scope chips and workspace labeling remain legible at smaller widths |
| Mutation restriction note visible | Non-loopback Hooks & Extensions environment | Restriction messaging fits the page hierarchy and does not overwhelm the route |

---

### Related Test Cases

- `TC-FUNC-008`, `TC-FUNC-009`, `TC-FUNC-010`, and `TC-FUNC-012` cover the same routes functionally.
- `TC-INT-011` and `TC-INT-013` validate scoped and transport-specific variants.

---

### Notes

- Include one screenshot for dialogs and one for the workspace-scoped MCP view if visual drift is observed.
