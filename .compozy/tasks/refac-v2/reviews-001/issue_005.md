---
status: resolved
file: internal/store/globaldb/global_db_session.go
line: 124
severity: high
author: claude-code
provider_ref:
---

# Issue 005: ReconcileSessions orphans every session not in input

## Review Comment

`loadSessionIDs` loads every session in the database with no state filter. If `ReconcileSessions` is called with a partial list of sessions (e.g., only sessions for one workspace), every other session in the entire database is marked `orphaned`. If two callers reconcile different workspaces, they orphan each other's sessions. The `UPDATE` at line 177 unconditionally sets `state = 'orphaned'` for any session ID not in the input `seen` set.

```go
existing, err := g.loadSessionIDs(ctx, tx) // loads ALL sessions
// ...
for id := range existing {
    if _, ok := seen[id]; ok { continue }
    // marks as orphaned — even sessions from other workspaces
    tx.ExecContext(ctx, `UPDATE sessions SET state = ? ...`, "orphaned", ...)
}
```

**Fix:** Either scope `loadSessionIDs` to the relevant workspace (e.g., `WHERE workspace_id = ?`), or only orphan sessions that were in a non-terminal state (e.g., `WHERE state NOT IN ('stopped', 'orphaned')`), or require the caller to pass the full session list for the entire database.

## Triage

- Decision: `invalid`
- Analysis: The current production call path is `Observer.Reconcile`, which always builds a full snapshot of every session metadata file under the sessions directory before calling `ReconcileSessions`. In that workflow, orphaning rows that are absent from the snapshot is the intended behavior.
- Analysis: Changing `ReconcileSessions` to orphan only a subset of rows would weaken the daemon's recovery semantics and hide genuinely missing sessions. There is no in-repo caller that passes a workspace-scoped partial list.
- Conclusion: This is a hypothetical misuse concern rather than a demonstrated bug in the current implementation, so no production change is warranted in this batch.
