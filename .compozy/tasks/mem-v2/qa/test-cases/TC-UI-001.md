# TC-UI-001: Knowledge View Reflects Memory Inventory

**Priority:** P2
**Status:** Not Run

## Preconditions

- Web UI is available and authenticated.
- Memory entries exist across at least one workspace.

## Steps

1. Open the Knowledge view.
2. Browse the current memory inventory.
3. Open one entry and compare it with API data.

**Expected:** Knowledge presents the same memory inventory exposed by the daemon APIs and does not hide scope or workspace_id details needed for operator workflows.

## Evidence To Capture

- Screenshot of the Knowledge view.
- Matching API payload for the selected memory.
