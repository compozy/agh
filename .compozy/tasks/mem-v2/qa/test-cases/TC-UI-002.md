# TC-UI-002: Memory Settings Matches Config Lifecycle

**Priority:** P2
**Status:** Not Run

## Preconditions

- Web UI Memory Settings page is reachable.
- Config changes can be inspected through daemon APIs.

## Steps

1. Open Memory Settings.
2. Verify displayed values for provider, workspace metadata, and dream/extractor settings.
3. Compare the displayed values to the config lifecycle source of truth.

**Expected:** Memory Settings reflects the canonical config lifecycle state and uses the same normalized enum values exposed by the daemon.

## Evidence To Capture

- Screenshot of Memory Settings.
- Matching config or settings API payload.
