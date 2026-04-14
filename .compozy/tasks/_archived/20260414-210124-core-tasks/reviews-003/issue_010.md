---
status: resolved
file: internal/extension/host_api_tasks.go
line: 456
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM565HzM,comment:PRRC_kwDOR5y4QM63qGak
---

# Issue 010: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Validate scope binding before resolving `workspace`.**

Both helpers resolve the workspace reference first, so `scope=global` or an empty/invalid scope can turn an invalid-params request into `workspace not found`, and that makes the response depend on whether the workspace exists.



Also applies to: 471-494

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/extension/host_api_tasks.go` around lines 434 - 456,
taskQueryFromParams currently resolves the workspace (resolveTaskWorkspaceID)
before validating the requested scope, which can convert
invalid-scope/empty-scope errors into "workspace not found"; instead, normalize
and validate the scope binding from params.Scope (call Scope.Normalize() and any
scope-binding validation you have) first and only call resolveTaskWorkspaceID
when the scope indicates a workspace-bound query. Update taskQueryFromParams to:
normalize/validate params.Scope, then conditionally call resolveTaskWorkspaceID
and set WorkspaceID; keep the existing validateTaskChannel and query.Validate
flow. Apply the same change to the other function in this file that also calls
resolveTaskWorkspaceID before scope validation (search for other usages of
resolveTaskWorkspaceID and fix them similarly).
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  Both `taskQueryFromParams` and `createTaskSpecFromRequest` resolve the workspace before validating scope binding, which can turn an invalid-scope request into a misleading workspace-not-found error. I will validate/normalize scope first and only resolve a workspace when the scope is workspace-bound.
  Resolution: Normalized and validated scope before workspace lookup, added `resolveTaskWorkspaceBinding(...)` for create flows, and added Host API regression coverage proving invalid scope/global binding errors win over workspace lookup failures.
