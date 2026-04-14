---
status: resolved
file: web/src/routes/_app/bridges.tsx
line: 206
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM56sg4W,comment:PRRC_kwDOR5y4QM63ZMIK
---

# Issue 016: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Don’t silently widen a workspace bridge to global.**

If the user chose workspace scope but `activeWorkspaceId` is missing, this creates a broader-scoped bridge than requested. That is a bad failure mode for a mutating action; it should fail fast with a validation error instead.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@web/src/routes/_app/bridges.tsx` around lines 205 - 206, The code currently
widens a user-selected "workspace" scope to "global" when activeWorkspaceId is
missing by computing scope from createDraft.scope and activeWorkspaceId;
instead, validate and fail fast: if createDraft.scope === "workspace" and
activeWorkspaceId is falsy, do not set scope to "global"—instead raise/return a
validation error (e.g., set form error, throw, or abort the submit/save flow) so
the bridge creation/update is blocked; update the logic around variable scope
(and any callers like the submit/save handler that uses scope) to perform this
check and surface a clear validation message to the user.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause: bridge creation silently widens a requested workspace-scoped mutation to global scope if the active workspace disappears while the dialog is open.
- Fix approach: fail fast with a user-visible validation error instead of broadening the scope.
