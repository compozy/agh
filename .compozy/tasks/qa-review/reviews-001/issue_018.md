---
status: resolved
file: web/src/hooks/routes/use-knowledge-page.ts
line: 8
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM59oaQ4,comment:PRRC_kwDOR5y4QM67VX7X
---

# Issue 018: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Avoid cross-system imports from `@/systems/knowledge/lib/*` in route hooks.**

This hook should consume knowledge APIs via the knowledge public barrel, not internals, to preserve module boundaries and avoid tight coupling to internal file layout.  
  
As per coding guidelines, "Cross-system imports MUST only go through the public barrel (`@/systems/<domain>`). Never reach into another system's internals".

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@web/src/hooks/routes/use-knowledge-page.ts` around lines 4 - 8, The hook
imports internal symbols knowledgeMemoryKey, filterKnowledgeMemories, and
sortKnowledgeMemories directly from "@/systems/knowledge/lib/*"; update the
imports to consume these APIs from the knowledge public barrel (import from
"@/systems/knowledge") instead to respect module boundaries, or if those symbols
are not yet exported from the public barrel, add/export them (or small wrapper
functions) from the public barrel so use-knowledge-page.ts can import
knowledgeMemoryKey, filterKnowledgeMemories, and sortKnowledgeMemories via
"@/systems/knowledge" instead of reaching into internal lib files.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - `use-knowledge-page.ts` currently reaches into `@/systems/knowledge/lib/*` instead of consuming helpers through the knowledge system's public barrel.
  - Root cause: `knowledgeMemoryKey`, `filterKnowledgeMemories`, and `sortKnowledgeMemories` are not exported from `web/src/systems/knowledge/index.ts`, so the route hook bypasses the system boundary.
  - Fix plan: export the needed helpers from the knowledge barrel and switch the hook imports to `@/systems/knowledge`, then keep the hook tests as the regression surface.
