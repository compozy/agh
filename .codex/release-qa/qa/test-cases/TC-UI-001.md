## TC-UI-001: Network Web UI Smoke

**Priority:** P1
**Type:** UI/Visual
**Status:** Not Run
**Estimated Time:** 10 minutes
**Created:** 2026-04-24
**Last Updated:** 2026-04-24

### Objective

Verify the browser-visible network surface loads and reflects live daemon data.

### Preconditions

- AGH web dev server or e2e web lane can start.
- Network fixture data is available through the daemon.

### Test Steps

1. Open the AGH web app network route.
   **Expected:** Network workspace is visible without console errors.

2. Select a channel and a peer.
   **Expected:** Header, peer count, message count, and timeline update for the selected room.

3. Trigger or inspect a send/composer flow if available.
   **Expected:** UI uses typed protocol kinds and does not obscure errors.

### Edge Cases & Variations

| Variation     | Input                          | Expected Result                          |
| ------------- | ------------------------------ | ---------------------------------------- |
| Empty network | No channels                    | Empty state is visible and non-crashing. |
| API failure   | Network endpoint returns error | Error state is visible and actionable.   |
