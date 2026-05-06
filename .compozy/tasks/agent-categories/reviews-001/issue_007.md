---
provider: coderabbit
pr: "113"
round: 1
round_created_at: 2026-05-06T20:42:04.329549Z
status: resolved
file: web/src/components/stories/app-sidebar.stories.tsx
line: 8
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM6AH9Ap,comment:PRRC_kwDOR5y4QM6-k_QI
---

# Issue 007: _🛠️ Refactor suggestion_ | _🟠 Major_ | _⚡ Quick win_
## Review Comment

_🛠️ Refactor suggestion_ | _🟠 Major_ | _⚡ Quick win_

**Use the agent system barrel for this type import.**

`AppSidebar` is outside the agent system, so importing `AgentPayload` from `@/systems/agent/types` reaches into another system's internals. Please switch this to the public barrel to keep the boundary stable.

<details>
<summary>Suggested import change</summary>

```diff
-import type { AgentPayload } from "@/systems/agent/types";
+import type { AgentPayload } from "@/systems/agent";
```
</details>

 

As per coding guidelines, `Cross-system imports: only through the public barrel (`@/systems/`<domain>), never reach into another system's internals`.

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
import { storyAgentNames, storyWorkspacePaths } from "@/storybook/fintech-scenario";
import { agentFixtures } from "@/systems/agent/mocks";
import type { AgentPayload } from "@/systems/agent";
import { sessionFixtures } from "@/systems/session/mocks";
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against current code. Fix only still-valid issues, skip the
rest with a brief reason, keep changes minimal, and validate.

In `@web/src/components/stories/app-sidebar.stories.tsx` around lines 5 - 8,
Replace the direct internal type import of AgentPayload from
"@/systems/agent/types" with the public barrel export from "@/systems/agent";
update the import line that currently brings in AgentPayload so it imports from
"@/systems/agent" alongside or instead of agentFixtures (and keep using
storyAgentNames, storyWorkspacePaths, sessionFixtures as before) to respect
cross-system boundaries and use the agent system's public API surface.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - `web/src/components/stories/app-sidebar.stories.tsx` still imports `AgentPayload` from `@/systems/agent/types`.
  - Root cause: the story reaches into another system's internals even though `web/src/systems/agent/index.ts` already exports `AgentPayload`.
  - Fix approach: switch the story to the public agent-system barrel so the story respects the repo’s cross-system boundary rule.
