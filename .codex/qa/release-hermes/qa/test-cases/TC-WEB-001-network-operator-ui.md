# TC-WEB-001: Network Operator UI

**Priority:** P0
**Type:** UI / E2E
**Status:** Pass
**Created:** 2026-04-24

## Objective

Verify that the web UI exposes the operator-critical network state: channels, local peers, remote peers, timeline events, reload continuity, and status/error states.

## Preconditions

- Web app and daemon test server can run locally.
- Browser automation is available.
- Network route test data is seeded by the e2e harness.

## Test Steps

1. Open the web application network route.
   **Expected:** route loads with no console/runtime failures.

2. Create or inspect a channel and peers.
   **Expected:** channel name, peer identity and capability information are visible.

3. Send or replay network timeline events.
   **Expected:** timeline shows sent/received/delivered/rejected rows with stable ordering.

4. Reload the page.
   **Expected:** channel and timeline state remain visible from persisted API state.

5. Capture desktop and mobile screenshots if the route is manually exercised.
   **Expected:** no overlapping text, broken controls, or inaccessible critical actions.

## Execution History

| Date       | Tester | Build | Result | Notes                                                                                                                                                                     |
| ---------- | ------ | ----- | ------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 2026-04-24 | Codex  | local | Pass   | Full daemon-served Playwright suite passed 15/15 specs. Network spec passed: create channel, inspect peers, observe timeline state, and reload without losing visibility. |
