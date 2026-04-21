---
status: resolved
file: web/src/storybook/web-storybook-stories-and-fixtures.test.tsx
line: 18
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM575wG1,comment:PRRC_kwDOR5y4QM65CLB5
---

# Issue 004: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Avoid deep cross-system imports in this test.**

Line 13–18 imports internal story modules directly from other systems, which couples this test to internals instead of public system boundaries.



As per coding guidelines, "Only import from cross-system dependencies through the public barrel export (`@/systems/<domain>`), never reach into another system's internals".

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@web/src/storybook/web-storybook-stories-and-fixtures.test.tsx` around lines
13 - 18, The test is importing story modules via deep cross-system paths
(knowledge-detail-panel.stories, knowledge-list-panel.stories,
network-channels-list-panel.stories, copy-button.stories, read-content.stories,
search-content.stories); change these to use each system's public barrel export
(e.g. import from "@/systems/knowledge", "@/systems/network",
"@/systems/session") so the test depends only on the public system boundary, and
if those barrels do not currently re-export the story fixtures, add named
exports for the relevant story modules in the corresponding system barrel files
(export { default as KnowledgeDetailStories } from
"./components/stories/knowledge-detail-panel.stories", etc.) then update the
test imports to import the story symbols from the public barrels.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `invalid`
- Notes:
  - Verified the current imports and the public barrels in `web/src/systems/knowledge/index.ts`, `web/src/systems/network/index.ts`, and `web/src/systems/session/index.ts`.
  - The test is intentionally a Storybook regression harness for specific story modules. Re-exporting story files through public system barrels would widen those runtime public APIs with Storybook-only modules purely to satisfy a test import style.
  - That change would couple production system entry points to non-production story modules and requires additional out-of-scope barrel edits with no user-facing or behavioral benefit.
  - The direct story-module imports are the correct boundary for this test’s purpose, so this finding is not actionable.
  - Analysis complete; no code change was warranted.
