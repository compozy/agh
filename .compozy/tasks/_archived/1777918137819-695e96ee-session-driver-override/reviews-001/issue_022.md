---
status: resolved
file: web/src/systems/session/components/session-create-dialog.tsx
line: 79
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM581a0J,comment:PRRC_kwDOR5y4QM66RFP6
---

# Issue 022: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Keep the dialog from closing while submit is in flight.**

`showCloseButton={!isSubmitting}` only hides the X. The close path is still wired straight through `onOpenChange`, so an overlay click or Escape can dismiss the dialog mid-submit and hide any eventual error.


<details>
<summary>Suggested change</summary>

```diff
-    <Dialog onOpenChange={onOpenChange} open={open}>
+    <Dialog
+      open={open}
+      onOpenChange={nextOpen => {
+        if (isSubmitting && !nextOpen) return;
+        onOpenChange(nextOpen);
+      }}
+    >
```
</details>

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
    <Dialog
      open={open}
      onOpenChange={nextOpen => {
        if (isSubmitting && !nextOpen) return;
        onOpenChange(nextOpen);
      }}
    >
      <DialogContent
        className="gap-0 p-0 text-[color:var(--color-text-primary)] sm:max-w-[30rem]"
        data-testid="session-create-dialog"
        showCloseButton={!isSubmitting}
      >
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@web/src/systems/session/components/session-create-dialog.tsx` around lines 74
- 79, The dialog can still be dismissed via overlay/Escape while a submit is in
flight because onOpenChange is passed directly; wrap onOpenChange with a guarded
handler (e.g., handleOpenChange) that ignores attempts to close when
isSubmitting is true and otherwise forwards the new open value, then pass that
handler to the Dialog component (leave open, isSubmitting, DialogContent, and
showCloseButton as-is).
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - The review is correct. `showCloseButton={!isSubmitting}` only removes the close affordance, but the dialog root still forwards overlay and Escape dismissals through `onOpenChange`.
  - Root cause: submit-state guarding is applied to visible controls only, not to the dialog close channel itself.
  - Fix approach: introduce a guarded open-change handler that ignores close attempts while `isSubmitting` is true, and add regression coverage for submit-in-flight dismissal behavior. This requires a small scope exception in `web/src/systems/session/components/session-create-dialog.test.tsx` because the scoped production file has an existing component test suite there.
  - Resolved: the dialog now routes all close attempts through a guarded handler, so backdrop/Escape dismissals are ignored while submission is in flight; the component test suite now covers both blocked and allowed backdrop dismissals.
  - Verified: focused Vitest session tests passed, then `make web-lint`, `make web-typecheck`, `make web-test`, and `make verify` all completed successfully.
