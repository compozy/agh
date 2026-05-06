---
provider: coderabbit
pr: "113"
round: 1
round_created_at: 2026-05-06T20:42:04.329549Z
status: resolved
file: web/src/systems/session/components/session-create-dialog.tsx
line: 119
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM6AH9A_,comment:PRRC_kwDOR5y4QM6-k_Qk
---

# Issue 012: _⚠️ Potential issue_ | _🟡 Minor_ | _⚡ Quick win_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_ | _⚡ Quick win_

**Disable the agent picker when no workspace is active.**

This control stays enabled whenever `agents.length > 0`, even if `workspace === undefined` and the dialog is already in its “choose a workspace first” state. That leaves stale workspace data interactive even though the session cannot be started. Please fold `!workspaceSelected` into the disabled/placeholder logic here, and mirror the same guard on the provider picker for consistency.

 

As per coding guidelines, "Truthful UI > plausible UI — don't render controls or metrics the runtime doesn't actually support; when Paper artboards conflict with daemon truth, daemon wins".

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against current code. Fix only still-valid issues, skip the
rest with a brief reason, keep changes minimal, and validate.

In `@web/src/systems/session/components/session-create-dialog.tsx` around lines
111 - 119, The AgentCommandSelect is currently enabled when agents.length > 0
even if no workspace is active; update its disabled and placeholder logic to
also consider the workspaceSelected flag (i.e., disabled should be true when
!workspaceSelected || !hasAgents || isSubmitting) and change the placeholder to
something like "Select a workspace first" when !workspaceSelected; apply the
same guard and placeholder change to the provider picker component so both
pickers are disabled and show the workspace-required message when
workspaceSelected is false (refer to AgentCommandSelect,
trimmedSelectedAgentName, onAgentChange, hasAgents, isSubmitting and the
provider picker component name in the file).
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - `SessionCreateDialog` still enables the agent picker whenever agents exist, even if no workspace is selected; the provider select has the same problem.
  - Root cause: the disabled/placeholder logic only checks data availability and submission state, not whether the dialog is in a workspace-ready state.
  - Fix approach: gate both pickers on `workspaceSelected`, surface a "select a workspace first" placeholder/message, and extend dialog tests to cover the truthful disabled state.
