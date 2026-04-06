---
status: pending
file: internal/httpapi/workspaces.go
line: 107
severity: high
author: claude-code
provider_ref:
---

# Issue 002: getWorkspace loads all sessions then filters in memory

## Review Comment

The `getWorkspace` handler calls `h.sessions.ListAll(ctx)` to fetch every session in the database, then filters the result in memory via `filterSessionInfosByWorkspaceID`. As the session count grows, this becomes O(N) in total sessions for every workspace detail request.

The same pattern appears in `udsapi/workspaces.go:107`.

The `SessionListQuery` in `store/global_db.go:324` already builds SQL where-clauses from `state` and `agent_name`, but does not support a `workspace_id` filter. Adding a `WorkspaceID` field to `SessionListQuery` and wiring it through `ListSessions` SQL would push the filtering to SQLite where it belongs.

**Suggested fix:** Add `WorkspaceID string` to `store.SessionListQuery`, add a `stringClause("workspace_id", query.WorkspaceID)` to the clause builder in `ListSessions`, and use it from the `getWorkspace` handler instead of fetching all and filtering.

## Triage

- Decision: `UNREVIEWED`
- Notes:
