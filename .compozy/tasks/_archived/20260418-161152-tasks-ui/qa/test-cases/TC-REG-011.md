## TC-REG-011: Settings shell navigation and section availability

**Priority:** P0
**Type:** Regression
**Status:** Not Run
**Estimated Time:** 12 minutes
**Created:** 2026-04-17
**Last Updated:** 2026-04-17
**Module:** Settings Shell
**Route / Surface:** `/_app/settings*` if present
**Design Reference:** `docs/design/paper/settings/AGH Settings — General@2x.png`, `docs/design/paper/settings/AGH Settings — Providers@2x.png`, `docs/design/paper/settings/AGH Settings — Automation@2x.png`, `docs/design/paper/settings/AGH Settings — Network@2x.png`, `docs/design/paper/settings/AGH Settings — Observability@2x.png`, `docs/design/paper/settings/AGH Settings — Memory@2x.png`, `docs/design/paper/settings/AGH Settings — Skills@2x.png`, `docs/design/paper/settings/AGH Settings — Hooks & Extensions@2x.png`, `docs/design/paper/settings/AGH Settings — MCP Servers@2x.png`, `docs/design/paper/settings/AGH Settings — Environments@2x.png`
**Execution Lane:** Manual + browser regression if preflight passes

### Objective

Verify the Settings shell is reachable, section navigation is stable, and the planned settings subsections are visible without breaking the main operator app shell.

### Preconditions

- [ ] `TC-REG-010` passed.
- [ ] The running app exposes a Settings entrypoint.
- [ ] The selected workspace or global scope is valid for settings navigation.

### Test Steps

1. Open the Settings surface from the shipped app shell entrypoint.
   **Expected:** The Settings shell loads inside the operator app rather than as a detached or broken route.
2. Navigate through the available Settings sections.
   **Expected:** The section list is stable and the page updates without breaking shell chrome or workspace context.
3. Compare the visible section set to the local Paper exports.
   **Expected:** The shipped sections match the intended operator information architecture closely enough for downstream CRUD/save tests.
4. Reload the current Settings route.
   **Expected:** The shell reload preserves the active section or resolves to a coherent default section.

### Edge Cases & Variations

| Variation | Input / State | Expected Result |
| --- | --- | --- |
| Missing section | one Paper-exported section is absent | document the gap as a discrepancy or blocker for the affected downstream case |
| Deep-link route | section opened directly by URL | the shell still renders and the correct section is active |
| Workspace/global switch | scope changes during Settings navigation | the shell remains stable and the active section updates cleanly |

### Related Test Cases

- `TC-REG-010`
- `TC-REG-012`
- `TC-REG-013`
- `TC-REG-014`

### Notes

- This case is part of Smoke when Settings is present, because all deeper Settings flows depend on shell stability first.
