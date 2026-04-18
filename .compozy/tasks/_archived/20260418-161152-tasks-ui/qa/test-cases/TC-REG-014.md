## TC-REG-014: Settings advanced scoped configuration flow

**Priority:** P1
**Type:** Regression
**Status:** Not Run
**Estimated Time:** 18 minutes
**Created:** 2026-04-17
**Last Updated:** 2026-04-17
**Module:** Settings Advanced Configuration
**Route / Surface:** first shipped advanced scope-aware flow, preferring MCP Servers then Hooks & Extensions
**Design Reference:** `docs/design/paper/settings/AGH Settings — MCP Servers@2x.png`, `docs/design/paper/settings/AGH Settings — Hooks & Extensions@2x.png`
**Execution Lane:** Manual + targeted browser regression if preflight passes

### Objective

Verify one advanced Settings surface handles scoped configuration, validation, and persistence for a non-trivial operator workflow such as MCP or hooks/extensions management.

### Preconditions

- [ ] `TC-REG-010` passed.
- [ ] An advanced scope-aware Settings surface exists on the execution branch.
- [ ] Any required workspace or credentials prerequisites are safe test fixtures.

### Test Steps

1. Open the chosen advanced Settings surface.
   **Expected:** The page shows advanced configuration controls rather than a generic placeholder.
2. Create or edit one scoped configuration entry.
   **Expected:** The UI accepts valid input, validates required fields, and saves without shell breakage.
3. Verify the saved configuration remains visible after navigation or reload.
   **Expected:** Persistence and scope labeling remain coherent.
4. Remove or revert the change if cleanup is needed.
   **Expected:** The environment returns to a known state without hidden residual config.

### Edge Cases & Variations

| Variation | Input / State | Expected Result |
| --- | --- | --- |
| Workspace-scoped config | config attached to one workspace only | scope labels and visibility remain accurate |
| Invalid config payload | malformed advanced setting | field or form validation blocks save with explicit feedback |
| Partial data | optional fields omitted | the surface handles missing optional input without corrupting the saved entry |

### Related Test Cases

- `TC-REG-011`
- `TC-REG-013`

### Notes

- This case exists so task_19 does not reduce Settings coverage to a shell-only smoke pass.
