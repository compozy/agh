---
provider: coderabbit
pr: "113"
round: 1
round_created_at: 2026-05-06T20:42:04.329549Z
status: resolved
file: web/src/systems/agent/components/agent-category-tree.tsx
line: 57
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM6AH9Aw,comment:PRRC_kwDOR5y4QM6-k_QR
---

# Issue 008: _⚠️ Potential issue_ | _🟠 Major_ | _⚡ Quick win_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_ | _⚡ Quick win_

**Don't collapse fetch errors into the empty/install state.**

This branch shows the same "Run `agh install`" message for both `agentsError` and an actually empty agent list. That will mislead users on request failures, and it also drops any stale-but-usable `agents` data if the query errors during a background refresh. Please render a distinct error state here, or at least only treat it as empty when `!agentsError && agents.length === 0`.

 

As per coding guidelines, `Handle all states — loading, error, and empty (never assume data exists)`.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against current code. Fix only still-valid issues, skip the
rest with a brief reason, keep changes minimal, and validate.

In `@web/src/systems/agent/components/agent-category-tree.tsx` around lines 47 -
57, The current render collapses fetch errors into the empty/install state: when
agentsError is truthy we still show the "Run `agh install`" message and drop any
existing agents; update the render logic in agent-category-tree (check
agentsError and agents) to separate error vs empty states—if agentsError render
a distinct error UI (or at minimum return the error state when agentsError is
true), otherwise if !agentsError && (!agents || agents.length === 0) render the
empty/install message; use the existing symbols agentsError and agents (and the
component AgentCategoryTree) to locate where to branch the UI and ensure you do
not discard stale agents when agentsError is set during background refresh.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - `AgentCategoryTree` still collapses `agentsError` and the empty list into the same `agents-empty` install message.
  - Root cause: the current early-return branch treats any fetch error as equivalent to "no agents installed", and it would also hide stale usable data when `agents` is still present.
  - Fix approach: separate loading, error, and empty states; keep rendering the tree when `agents` is available, render a dedicated error message when it is not.
