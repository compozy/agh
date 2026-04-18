---
status: resolved
file: web/src/systems/automation/components/stories/automation-editor-dialog.stories.tsx
line: 27
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM575wG5,comment:PRRC_kwDOR5y4QM65CLB9
---

# Issue 006: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Avoid page-level orchestration inside this story harness.**

Using `useAutomationPage()` here couples the story to route/page orchestration and makes the story less isolated. Prefer a local, deterministic harness that passes explicit dialog props.

<details>
<summary>Proposed refactor</summary>

```diff
-import { useAutomationPage } from "@/hooks/routes/use-automation-page";
 import { createAutomationJobDraft } from "@/systems/automation";
@@
 function AutomationEditorDialogHarness() {
-  const page = useAutomationPage();
-  const activeWorkspaceId = page.editorDialogProps.activeWorkspaceId;
+  const activeWorkspaceId = "workspace-storybook";
   const [draft, setDraft] = useState(() => createAutomationJobDraft(activeWorkspaceId));
 
   return (
     <AutomationEditorDialog
-      {...page.editorDialogProps}
+      activeWorkspaceId={activeWorkspaceId}
       editor={{
         draft,
         isPending: false,
```
</details>



As per coding guidelines, "`web/src/**/*.tsx`: UI components MUST be pure and presentational; orchestration logic lives in pages/routes".

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
function AutomationEditorDialogHarness() {
  const activeWorkspaceId = "workspace-storybook";
  const [draft, setDraft] = useState(() => createAutomationJobDraft(activeWorkspaceId));

  return (
    <AutomationEditorDialog
      activeWorkspaceId={activeWorkspaceId}
      editor={{
        draft,
        isPending: false,
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In
`@web/src/systems/automation/components/stories/automation-editor-dialog.stories.tsx`
around lines 20 - 27, The AutomationEditorDialogHarness currently imports
page-level orchestration via useAutomationPage() and pulls
page.editorDialogProps (and activeWorkspaceId) into state; replace that with a
local, deterministic harness: remove useAutomationPage() usage in
AutomationEditorDialogHarness, call createAutomationJobDraft with a fixed test
workspace id (instead of activeWorkspaceId), and construct explicit dialog props
(handlers, open flag, initial values) inline in the harness to pass into
<AutomationEditorDialog /> rather than spreading page.editorDialogProps; keep
references to AutomationEditorDialogHarness, createAutomationJobDraft, and
page.editorDialogProps to locate the code and ensure mock callbacks
(onClose/onSave) are provided so the component remains fully presentational and
isolated for story usage.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - Verified in `web/src/systems/automation/components/stories/automation-editor-dialog.stories.tsx`.
  - The story harness currently calls `useAutomationPage()` and spreads `page.editorDialogProps` into the component, which pulls route/page orchestration into a Storybook story for a presentational dialog.
  - Root cause: the story reuses page-level state instead of constructing a deterministic local editor state for the dialog.
  - Fix approach: remove `useAutomationPage()`, create a fixed story workspace id locally, and pass explicit dialog props plus local editor state so the story remains isolated and presentational.
  - Resolved by replacing the page hook with a local harness and locking that regression down in the scoped Storybook test.
