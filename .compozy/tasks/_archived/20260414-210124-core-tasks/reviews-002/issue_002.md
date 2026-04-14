---
status: resolved
file: internal/api/core/tasks.go
line: 705
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM564LfE,comment:PRRC_kwDOR5y4QM63o2Op
---

# Issue 002: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Validate `scope` before resolving `workspace`.**

These paths look up the workspace even when the normalized scope may be `global`. That makes an invalid request return `404` when the workspace ref does not exist and `400` when it does, which both leaks workspace existence and makes status codes depend on unrelated data. Only resolve the workspace ref after you know the request is actually workspace-scoped.




Also applies to: 738-741, 764-767

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/api/core/tasks.go` around lines 699 - 705, The handler currently
resolves workspaceRef via h.lookupWorkspaceID whenever a workspace query param
is supplied, which can leak workspace existence; change the logic to first
inspect the normalized scope (the request's scope parameter or the code path
that sets query.Scope) and only call h.lookupWorkspaceID and set
query.WorkspaceID when the request is actually workspace-scoped (i.e., scope !=
"global" and matches the workspace-scoped enum/value); apply the same
conditional change to the other occurrences noted (the similar blocks around the
other occurrences for lines 738-741 and 764-767) so workspace resolution only
happens for workspace-scoped requests.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  `parseTaskListQuery`, `createTaskSpecFromRequest`, and `createChildTaskSpecFromRequest` resolve the workspace reference before validating the normalized scope/workspace binding. A request like `scope=global&workspace=missing` can therefore leak workspace existence via `404` instead of consistently failing with `400`.
  Root cause: workspace lookup is performed before the scope-specific validation path.
  Planned fix: validate the normalized scope first and only resolve workspace references for workspace-scoped requests; add handler tests that lock in the non-leaking `400` behavior.

## Resolution

- Reordered the task API request parsing so scope/workspace binding is validated before any workspace lookup occurs, and only workspace-scoped requests resolve workspace references.
- Added regression coverage in `internal/api/core/tasks_test.go` to prove `scope=global` plus `workspace=...` fails with `400` without touching the workspace lookup path.
