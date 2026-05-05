# TC-UI-007: Network Empty Loading And Error States

**Priority:** P1
**Type:** UI
**Module:** Web Network Workspace
**Requirement:** Network pages should handle absent data and API failures gracefully.

## Objective

Verify top-level and room-level loading, disabled, empty, and error states communicate useful information without crashing.

## Preconditions

- Network status query can be mocked as loading, disabled, error, and enabled.
- Channel and peer detail/message queries can be mocked independently.

## Test Steps

1. Render route while status query is loading.
   **Expected:** Accessible loading spinner appears.
2. Render with status query error.
   **Expected:** Top-level error empty state appears with error message.
3. Render with `network.enabled=false`.
   **Expected:** Disabled network empty state appears.
4. Render enabled network with no rooms.
   **Expected:** Workspace shows select-room or no-match empty states.
5. Make selected room detail fail while status succeeds.
   **Expected:** Room-level error appears without replacing the whole workspace shell.
6. Render selected room with no messages.
   **Expected:** Timeline empty state appears.

## Edge Cases

| Variation | Input | Expected Result |
| --- | --- | --- |
| Message query loading with existing messages | refetch | existing messages remain visible |
| Peer detail 404 | selected peer gone | room error and stable sidebar |
| Channel messages 404 | history missing | room error state |

## Related

- SMOKE-001
- TC-UI-002
