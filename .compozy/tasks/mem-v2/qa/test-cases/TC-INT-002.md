# TC-INT-002: HTTP Search, Recall, and Snapshot Refresh

**Priority:** P1
**Status:** Not Run

## Preconditions

- HTTP API is reachable.
- A known workspace memory set exists.

## Steps

1. Search through HTTP and capture returned recall metadata.
2. Trigger reload or frozen snapshot invalidation.
3. Re-run search and verify the refreshed state is visible.

**Expected:** HTTP search returns stable recall data, memory_recall_signals stay observable, and frozen snapshot refresh makes new state visible without corrupting results.

## Required Evidence

- HTTP search payload before refresh.
- HTTP reload payload.
- HTTP search payload after refresh.
- Any memory_recall_signals evidence collected during the run.
