---
status: resolved
file: internal/httpapi/workspaces.go
line: 170
severity: high
author: claude-code
provider_ref:
---

# Issue 001: Workspace delete fails with opaque 500 on FK violation

## Review Comment

When deleting a workspace that has sessions referencing it, the SQLite foreign key constraint (`workspace_id TEXT NOT NULL REFERENCES workspaces(id)`) causes a `FOREIGN KEY constraint failed` error. This error is not recognized by `mapWorkspaceConstraintError` in `store/global_db.go:1083` (which only maps `UNIQUE CONSTRAINT` errors), nor by `statusForWorkspaceError` in `apisupport/session_workspace.go:104` (which doesn't handle FK errors), so users get a cryptic 500 Internal Server Error.

The same issue affects `udsapi/workspaces.go:170`.

**Suggested fix:** Before deleting, check whether sessions exist for the workspace and return a clear 409 Conflict error. Alternatively, add FK error detection in `mapWorkspaceConstraintError`:

```go
case strings.Contains(message, "foreign key constraint failed"):
    return ErrWorkspaceHasSessions // new sentinel
```

Then handle it in `statusForWorkspaceError` to return 409.

## Triage

- Decision: `valid`
- Root cause: workspace deletion flows through `workspace.Resolver.Unregister()` into `store.GlobalDB.DeleteWorkspace()`, which returns the raw SQLite foreign-key error when `sessions.workspace_id` still references the workspace. That raw error is not mapped to a workspace-domain sentinel, so the HTTP transport falls through to a 500.
- Fix plan: add a dedicated workspace sentinel for "workspace has sessions", map SQLite FK violations to it in the store delete path, and map that sentinel to HTTP 409 conflict.
- Scope note: this needs minimal supporting edits outside the four primary batch files in `internal/store/global_db.go`, `internal/apisupport/session_workspace.go`, and related tests because the root cause lives below the handler.

## Resolution

- Added `workspace.ErrWorkspaceHasSessions`, mapped SQLite foreign-key failures to it in the global store delete path, and mapped the sentinel to HTTP `409 Conflict`.
- Added regression coverage for the store delete path and the HTTP delete handler.
- Verified with `go test ./internal/workspace ./internal/store ./internal/observe ./internal/httpapi` and `make verify`.
