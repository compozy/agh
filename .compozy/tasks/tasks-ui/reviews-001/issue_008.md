---
status: resolved
file: internal/api/core/tasks_surface_internal_test.go
line: 219
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM575lbw,comment:PRRC_kwDOR5y4QM65B8fN
---

# Issue 008: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

**This case never reaches the workspace-resolution error path.**

The request only sets `lane=bogus`, so `taskInboxDomainQuery` fails validation before any workspace lookup happens. The `ErrWorkspaceNotFound` stub is dead code here; add a separate case with a valid lane plus `workspace=...` if you want coverage for workspace lookup failures.


As per coding guidelines, `**/*_test.go`: Focus on critical paths: workflow execution, state management, error handling.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/api/core/tasks_surface_internal_test.go` around lines 195 - 219, The
test currently exercises validation failure only (ParseTaskInboxQuery with
lane=bogus) so the workspaceServiceStub returning
workspacepkg.ErrWorkspaceNotFound is never reached; add a separate test case
that uses ParseTaskInboxQuery with a valid lane value and a workspace query
parameter (e.g., workspace=some-id) so that handlers.taskInboxDomainQuery is
invoked past validation and the workspaceServiceStub (configured to return
workspacepkg.ErrWorkspaceNotFound) triggers the workspace-resolution error path;
update assertions to expect a non-nil error and verify it matches
ErrWorkspaceNotFound (using the same assertTaskValidationError or a new
assertion for workspace resolution).
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes: Confirmed. The current request only sets `lane=bogus`, so `taskInboxDomainQuery` fails lane validation before any workspace lookup occurs. The configured `ErrWorkspaceNotFound` stub is dead code in this case. I’ll add a separate valid-lane request with `scope` and `workspace` so the workspace-resolution error path is actually exercised.
