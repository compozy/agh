---
status: resolved
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

- Decision: `invalid`
- Reasoning: the review comment’s proposed fix targets `store.GlobalDB.ListSessions`, but `httpapi.getWorkspace` does not read from the global SQLite session index. It calls `session.Manager.ListAll()`, which enumerates active sessions plus on-disk session metadata to preserve the API’s current source of truth, including workspace-path enrichment.
- Reasoning: adding `WorkspaceID` to `store.SessionListQuery` would not change this handler’s behavior without a broader contract change across the session manager and both API servers. That broader redesign is not justified by a concrete correctness bug in this batch.
- Follow-up note: if workspace-detail performance becomes a measured bottleneck, it should be addressed as a separate architectural change by introducing a dedicated session query surface rather than by patching an unused store query path.

## Resolution

- Marked invalid after confirming the current HTTP implementation does not use the store query path named in the review comment.
- No production code change was made for this issue.
- Final repository verification still passed with `make verify`.
