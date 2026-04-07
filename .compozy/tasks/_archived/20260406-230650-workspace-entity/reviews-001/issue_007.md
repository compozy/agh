---
status: resolved
file: internal/workspace/resolver.go
line: 448
severity: medium
author: claude-code
provider_ref:
---

# Issue 007: Workspace names starting with ws_ or ws- are unresolvable by name

## Review Comment

`lookupWorkspace` routes any identifier starting with `ws_` or `ws-` to `GetWorkspace` (ID lookup) and never falls back to name lookup:

```go
case strings.HasPrefix(target, "ws_"), strings.HasPrefix(target, "ws-"):
    ws, err := r.store.GetWorkspace(ctx, target)
```

Neither `Register` nor `Update` validate workspace names against these prefixes. A user can successfully create a workspace named `ws_alpha`, but subsequently `Resolve("ws_alpha")`, `Get("ws_alpha")`, and `session new --workspace ws_alpha` will all attempt an ID lookup, fail with "workspace not found", and never try the name lookup path.

**Suggested fix:** Either:
1. Validate workspace names in `Register`/`Update` to reject names starting with `ws_` or `ws-` (with a clear error message), or
2. Fall back to `GetWorkspaceByName` when `GetWorkspace` (ID lookup) returns `ErrWorkspaceNotFound`

Option 1 is simpler and prevents the ambiguity entirely.

## Triage

- Decision: `valid`
- Root cause: `lookupWorkspace()` treats any identifier starting with `ws_` or `ws-` as an ID-only lookup. If no workspace with that ID exists, it returns immediately instead of retrying the same value as a workspace name.
- Fix plan: keep the fast-path ID lookup for real workspace IDs, but fall back to name lookup when the ID path returns `ErrWorkspaceNotFound`. That fixes already-created workspaces whose names use the reserved-looking prefix without broadening the handler surface.

## Resolution

- Added a name-lookup fallback when a `ws_` or `ws-` identifier misses as an ID.
- Added resolver coverage proving `Resolve("ws_alpha")` succeeds for a workspace whose name starts with the ID prefix even when no workspace ID matches that string.
- Verified with targeted package tests and `make verify`.
