# TC-UI-005: Network Details Panel Tabs

**Priority:** P2
**Type:** UI
**Module:** Web Network Workspace
**Requirement:** Room details should expose about, members, and wire metadata without losing active room context.

## Objective

Verify details panel tab switching, capability rendering, member list, kind counts, and open/closed behavior.

## Preconditions

- Active channel has purpose, members, messages, and kind counts.
- Active peer has capability brief and rich catalog detail.

## Test Steps

1. Open details panel on a channel room.
   **Expected:** About tab shows purpose, created timestamp, and room activity.
2. Click Members tab.
   **Expected:** Member list shows local/remote chips and last-seen metadata.
3. Click Wire tab.
   **Expected:** Workspace, created-by, sessions, peers, last activity, and kind counts render.
4. Switch to a peer with capabilities.
   **Expected:** About tab shows peer ID, channel, last seen, and capability summaries.
5. Close details panel.
   **Expected:** Panel is removed and route search stores `details=closed`.

## Edge Cases

| Variation | Input | Expected Result |
| --- | --- | --- |
| No members | empty array | empty text, no crash |
| Missing capability detail | brief only | summary still visible |
| Long identifiers | peer IDs | mono badges wrap or truncate without overlap |

## Related

- TC-FUNC-003
- TC-FUNC-005
