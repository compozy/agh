---
status: resolved
file: internal/extension/host_api_tasks.go
line: 611
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM575lb0,comment:PRRC_kwDOR5y4QM65B8fS
---

# Issue 017: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Workspace filtering here can be ignored or broadened unexpectedly.**

Unlike `taskQueryFromParams`, these helpers only resolve `WorkspaceID` when `scope == workspace`. A dashboard/inbox request that passes `workspace` without that explicit scope will either reject the parameter or silently skip the workspace filter, which can return cross-workspace data. Resolve the workspace ID whenever `workspace` is present, then let scope only control scope validation.



Also applies to: 613-642

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/extension/host_api_tasks.go` around lines 576 - 611, The workspace
filter is only being resolved when query.Scope == taskpkg.ScopeWorkspace which
lets callers pass a workspace while using a different scope and bypass the
workspace filter; change taskDashboardQueryFromParams (and the similar
taskInboxQueryFromParams) to always resolve and set query.WorkspaceID when
params.Workspace is non-empty: first TrimSpace params.Workspace, call
taskpkg.ValidateScopeBinding as before to validate the binding, then
unconditionally call h.resolveTaskWorkspaceID(ctx, workspaceRef) and assign the
resulting workspaceID to query.WorkspaceID (still return errors as currently
done); keep scope logic and query.Validate() intact so scope validation remains
separate from workspace ID resolution.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes: Confirmed by comparison with `taskQueryFromParams`. The dashboard/inbox Host API helpers only resolve `WorkspaceID` when `scope == workspace`, so callers cannot use a workspace filter unless they also set `scope`, even though the list helper accepts workspace-only filtering. I’ll align dashboard/inbox workspace resolution with the list behavior while still rejecting the invalid `global + workspace` combination.
