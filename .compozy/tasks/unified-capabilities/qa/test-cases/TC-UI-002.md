# TC-UI-002: Room Filtering Selection And URL Synchronization

**Priority:** P1
**Type:** UI
**Module:** Web Network Workspace
**Requirement:** Network room selection should stay in sync with URL search and local UI state.

## Objective

Verify sidebar search, room selection, auto-selection, back/forward navigation, starred channels, and kind filters work without URL loops or stale state.

## Preconditions

- Network workspace has at least two channels and two peers.
- Browser or Vitest route harness can inspect URL search params.
- Local storage can be controlled.

## Test Steps

1. Open `/network?channel=builders`.
   **Expected:** Builders channel is selected and corresponding room item has active state.
2. Click a peer room.
   **Expected:** URL changes to `peer=<peer_id>` and clears `channel`.
3. Enter sidebar search text that hides the selected room.
   **Expected:** UI remains stable and does not crash; selection changes only according to defined auto-selection logic.
4. Toggle kind filter to `direct`.
   **Expected:** URL search stores `kind=direct`; timeline displays direct messages only.
5. Use browser back.
   **Expected:** Prior selection and filter restore.

## Edge Cases

| Variation | Input | Expected Result |
| --- | --- | --- |
| Invalid kind search | `kind=bad` | validation removes kind |
| Both peer and channel search | both params | peer takes precedence |
| Corrupt local storage | invalid JSON | ignored and reset to empty |

## Related

- TC-UI-101
- TC-FUNC-006
