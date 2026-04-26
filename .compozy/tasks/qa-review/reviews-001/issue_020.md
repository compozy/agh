---
status: resolved
file: web/src/hooks/routes/use-knowledge-page.ts
line: 113
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM59oaQ5,comment:PRRC_kwDOR5y4QM67VX7Y
---

# Issue 020: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**This `useEffect` is handling local state orchestration and should be removed.**

The reset behavior should be handled in explicit event paths (tab/search/selection mutations) rather than effect-driven local bookkeeping.  
  
As per coding guidelines, "`useEffect` is an escape hatch — only for external system sync; never for derived state or event responses".

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@web/src/hooks/routes/use-knowledge-page.ts` around lines 101 - 113, Remove
the effect block that watches deleteTargetKey/isDeletePending and instead clear
the delete state in the explicit event handlers; delete the useEffect that
references deleteTargetKey, isDeletePending, resetDeleteMutation,
setDeleteTargetKey, selectedMemory and knowledgeMemoryKey, and call
resetDeleteMutation() and setDeleteTargetKey(null) directly from the tab switch
handler, search-reset handler, and any selection/mutation handlers (where
selectedMemory changes or a memory is removed) so the reset happens as part of
those event paths rather than via an effect.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - The delete-target cleanup in `useKnowledgePage()` is implemented with a `useEffect` that watches local state and mutation state.
  - Root cause: local UI bookkeeping is being expressed as an effect instead of explicit event-path transitions for tab changes, search changes, selection changes, and successful deletes.
  - Fix plan: remove the effect, reset delete state from the relevant event handlers, and extend the existing knowledge hook/route tests that already exercise failed-delete recovery.
