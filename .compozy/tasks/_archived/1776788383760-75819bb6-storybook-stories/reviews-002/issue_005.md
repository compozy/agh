---
status: resolved
file: web/src/systems/automation/components/stories/automation-editor-dialog.stories.tsx
line: 7
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM575wG2,comment:PRRC_kwDOR5y4QM65CLB6
---

# Issue 005: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Replace the relative import with the `@/*` alias.**

This import should use the web alias to stay consistent with the enforced import rule.

<details>
<summary>Proposed fix</summary>

```diff
-import { AutomationEditorDialog } from "../automation-editor-dialog";
+import { AutomationEditorDialog } from "@/systems/automation/components/automation-editor-dialog";
```
</details>



As per coding guidelines, "Use path alias `@/*` to map to `./src/*` for all imports".

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
import { AutomationEditorDialog } from "@/systems/automation/components/automation-editor-dialog";
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In
`@web/src/systems/automation/components/stories/automation-editor-dialog.stories.tsx`
at line 7, The import uses a relative path; replace it with the project
path-alias form so it follows the enforced rule. Update the import of
AutomationEditorDialog (symbol: AutomationEditorDialog) to use the web alias
(start with "@/") pointing to the same module path under src (e.g.
"@/systems/automation/components/automation-editor-dialog") and ensure any
linting/tsconfig alias settings resolve correctly.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - Verified in `web/src/systems/automation/components/stories/automation-editor-dialog.stories.tsx`.
  - The story imports `AutomationEditorDialog` through a relative path while the applicable `web/AGENTS.md` requires `@/*` aliases for `web/src/*` imports.
  - Root cause: the story file was authored with a local relative import instead of the enforced project alias.
  - Fix approach: switch the component import to the existing `@/systems/automation/components/automation-editor-dialog` alias path and cover it in the regression test.
  - Resolved by switching to the alias import and asserting that import path in the scoped Storybook regression test.
