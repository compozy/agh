---
status: resolved
file: web/src/systems/automation/components/stories/automation-editor-dialog.stories.tsx
line: 23
severity: major
author: coderabbitai[bot]
provider_ref: review:4133289577,nitpick_hash:d261b49ca198
review_hash: d261b49ca198
source_review_id: "4133289577"
source_review_submitted_at: "2026-04-18T02:22:01Z"
---

# Issue 019: Remove useEffect and pass initial dialog state directly to the harness.
## Review Comment

`handleCreate` (line 241 in use-automation-page.ts) only calls `setEditor()` — pure state setup, not external system synchronization. Using `useEffect` to trigger this on mount violates the guideline: *"useEffect is an escape hatch — only use for external system synchronization; never for derived state or event responses."*

Instead, initialize the editor state in the harness directly:

```typescript
function AutomationEditorDialogHarness() {
const page = useAutomationPage();
const editorDialogProps = {
...page.editorDialogProps,
editor: {
draft: createAutomationJobDraft(page.activeWorkspaceId),
kind: "jobs" as const,
mode: "create" as const,
isPending: false,
onCancel: () => {},
onChange: () => {},
onSubmit: () => {},
},
};

return <AutomationEditorDialog {...editorDialogProps} />;
}
```

## Triage

- Decision: `VALID`
- Notes:
  - The harness uses a mount-time `useEffect` plus a `useRef` guard solely to call `page.handleCreate()`, which is a pure local state transition rather than external-system synchronization.
  - That is exactly the `useEffect` misuse prohibited by the scoped React guidance for web files.
  - Fix approach: initialize the dialog props directly in the harness with a create-mode editor state, preserving the same visible story behavior without effect-driven setup.
