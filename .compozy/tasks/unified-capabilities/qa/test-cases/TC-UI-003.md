# TC-UI-003: Create Channel Dialog Validation And Accessibility

**Priority:** P1
**Type:** UI
**Module:** Web Network Workspace
**Requirement:** Channel creation UI should prevent invalid submissions and expose clear accessible controls.

## Objective

Verify create-channel dialog input handling, agent selection, submit disabled state, loading state, error handling, and keyboard/form behavior.

## Preconditions

- Network route has active workspace and agents.
- Network status is enabled.
- Create-channel mutation can be mocked for success and failure.

## Test Steps

1. Click `Channel` button.
   **Expected:** Dialog opens with channel name, purpose, agent list, cancel, and submit controls.
2. Leave required fields empty.
   **Expected:** Submit is disabled.
3. Enter channel name and purpose, select one agent.
   **Expected:** Submit becomes enabled.
4. Submit the form.
   **Expected:** Mutation receives trimmed channel, purpose, active workspace ID, and selected agents; dialog closes on success.
5. Submit while mutation is pending.
   **Expected:** Loader appears and duplicate submit is prevented.
6. Trigger mutation error.
   **Expected:** Toast shows error and dialog remains available for correction.

## Edge Cases

| Variation | Input | Expected Result |
| --- | --- | --- |
| No active workspace | workspace missing | UI blocks or clearly errors before API call |
| No agents | empty agent list | empty state and submit disabled |
| Duplicate toggle | select then deselect | selected count updates |

## Related

- SMOKE-002
- TC-FUNC-008
