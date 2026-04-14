---
status: resolved
file: web/src/systems/network/components/network-create-channel-dialog.tsx
line: 55
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM56sg4b,comment:PRRC_kwDOR5y4QM63ZMIR
---

# Issue 022: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Use a real form submit path for the primary action.**

This dialog has a text input, but the primary action is only wired through `onClick`. Pressing Enter in the channel name field will not submit, which is a pretty common keyboard path for dialogs like this. 

<details>
<summary>Suggested fix</summary>

```diff
-        <div className="space-y-5 px-5 py-4">
+        <form
+          onSubmit={event => {
+            event.preventDefault();
+            onSubmit();
+          }}
+        >
+          <div className="space-y-5 px-5 py-4">
             ...
-        </div>
+          </div>
 
-        <DialogFooter className="border-[color:var(--color-divider)] bg-[color:var(--color-surface-panel)]">
+          <DialogFooter className="border-[color:var(--color-divider)] bg-[color:var(--color-surface-panel)]">
             <Button onClick={() => onOpenChange(false)} type="button" variant="outline">
               Cancel
             </Button>
             <Button
               data-testid="network-create-channel-submit"
               disabled={!canSubmit || isSubmitting}
-              onClick={onSubmit}
-              type="button"
+              type="submit"
             >
               {isSubmitting ? <Loader2 className="size-4 animate-spin" /> : null}
               Create Channel
             </Button>
-        </DialogFooter>
+          </DialogFooter>
+        </form>
```
</details>


Also applies to: 141-153

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@web/src/systems/network/components/network-create-channel-dialog.tsx` around
lines 43 - 55, The primary action in NetworkCreateChannelDialog is only wired to
an onClick handler so pressing Enter in the channel name input doesn't submit;
wrap the dialog content in a form (or add an onSubmit on the existing form
container), move the primary create logic into an onSubmit handler (e.g. reuse
the existing createChannel/create handler function referenced in the component)
and call event.preventDefault() as needed, and change the primary Button to
type="submit" so Enter in the TextInput triggers submission; apply the same fix
to the duplicate action referenced around the secondary block (lines 141-153) in
this component.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause: the dialog’s primary action is bound only to a click handler, so pressing Enter while focus is in the channel name input does not trigger submission. That is a real keyboard-accessibility gap for a dialog with a single primary action.
- Fix approach: move the action onto a real `<form>` submit path, keep validation in the existing submit handler, and change the primary button to `type="submit"`. Route coverage will be updated to assert Enter submits the dialog.
- Resolution: wrapped the dialog body/footer in a real form, moved submission to `onSubmit`, and converted the primary button to `type="submit"`.
- Verification: the network route test now submits via Enter, and the focused Vitest run plus `make web-lint`, `make web-typecheck`, and `make verify` passed.
