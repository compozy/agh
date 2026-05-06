# TC-UI-003: Session Inspector Surfaces Ledger and Recall Clues

**Priority:** P2
**Status:** Not Run

## Preconditions

- At least one session has memory activity.
- Session Inspector is available in the web UI.

## Steps

1. Open Session Inspector for a session with memory writes.
2. Inspect session replay, recall traces, and related artifacts.
3. Cross-check with ledger.jsonl and API responses.

**Expected:** Session Inspector exposes enough state to correlate memory activity with ledger.jsonl and daemon recall data.

## Evidence To Capture

- Session Inspector screenshots.
- Matching ledger.jsonl excerpt or API payload.
