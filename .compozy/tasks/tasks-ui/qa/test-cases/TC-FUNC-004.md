## TC-FUNC-004: Tasks kanban grouping and card actions

**Priority:** P1
**Type:** Functional
**Status:** Not Run
**Estimated Time:** 12 minutes
**Created:** 2026-04-17
**Last Updated:** 2026-04-17
**Module:** Tasks Kanban
**Route / Surface:** `/_app/tasks` in kanban mode
**Design Reference:** `docs/design/paper/tasks/AGH Tasks — Kanban View@2x.png`
**Execution Lane:** Manual + targeted browser regression

### Objective

Verify kanban mode groups tasks according to the shared grouping rules and preserves card actions and empty-column behavior.

### Preconditions

- [ ] The Tasks route is available.
- [ ] Seed data covers draft or pending, ready/in-progress, blocked, and failed or canceled states.
- [ ] At least one task in a visible column supports an action such as open, publish, or retry.

### Test Steps

1. Open Tasks and switch from list mode to kanban mode.
   **Expected:** The board renders distinct columns without losing the active workspace context.
2. Confirm task placement across columns.
   **Expected:** `draft` and `pending` appear in the Pending column, `failed` and `canceled` appear in the Failed column, and other statuses remain 1:1.
3. Open one task card action from the board.
   **Expected:** The action routes to the correct downstream flow and does not require switching back to list mode first.
4. Inspect at least one empty column.
   **Expected:** Empty columns remain visible and informative rather than collapsing the board layout.

### Edge Cases & Variations

| Variation | Input / State | Expected Result |
| --- | --- | --- |
| All items in one column | heavily skewed seed | the board still preserves all expected columns |
| Mixed failure states | failed plus canceled tasks | both map into the shared Failed column |
| Actionless card | card has no immediate mutation | open/detail navigation still works cleanly |

### Related Test Cases

- `TC-FUNC-003`
- `TC-FUNC-006`

### Notes

- This case is P1 because list/create/detail flows take precedence in Smoke, but it remains required for the full regression.
