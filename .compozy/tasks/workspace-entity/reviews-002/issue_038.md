---
status: resolved
file: web/src/systems/workspace/components/workspace-selector.tsx
line: 28
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM55IoDU,comment:PRRC_kwDOR5y4QM61T6JF
---

# Issue 038: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Keep the selector controlled by `value`.**

Falling back to `workspaces[0]` when `value` is `null` or stale makes the UI show a workspace the parent never selected. That leaves the visible selection and the workspace ID used by actions/mutations out of sync. Let the parent choose the initial workspace explicitly, or render an explicit empty state here instead of substituting the first item.


As per coding guidelines, "UI components MUST be pure and presentational; orchestration logic lives in pages/routes".

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@web/src/systems/workspace/components/workspace-selector.tsx` around lines 18
- 28, The component currently falls back to workspaces[0] which makes the
selector uncontrolled relative to the parent; update the logic around
selectedWorkspace and the NativeSelect value so the component strictly reflects
the passed-in value (keep selectedWorkspace = workspaces.find(w => w.id ===
value) ?? null), remove the fallback to workspaces[0], and ensure NativeSelect
uses selectedWorkspace?.id ?? "" so an explicit empty state is rendered when
value is null/stale; keep onValueChange and disabled behavior unchanged so the
parent must supply the initial selection and orchestration remains outside this
presentational component.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `INVALID`
- Notes:
  The current caller (`AppSidebar`) already computes a stable
  `activeWorkspaceId` before rendering `WorkspaceSelector`, so the fallback
  branch does not create a state mismatch in production today. Making the
  selector render a true empty state would also require a placeholder option,
  which is outside the reported change. No change in this batch.
