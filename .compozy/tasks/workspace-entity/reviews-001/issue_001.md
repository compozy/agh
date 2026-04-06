---
status: pending
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

- Decision: `UNREVIEWED`
- Notes:
