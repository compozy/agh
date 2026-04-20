---
status: resolved
file: packages/ui/src/components/combobox.tsx
line: 118
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM58EmKD,comment:PRRC_kwDOR5y4QM65P-uf
---

# Issue 012: _⚠️ Potential issue_ | _🔴 Critical_
## Review Comment

_⚠️ Potential issue_ | _🔴 Critical_

**`render` is wired to the wrong component.**

`Button` does not accept a `render` prop, so this composition will not type-check and would forward an invalid attribute if it slipped through. The `render` prop belongs on `ComboboxTrigger`/`ComboboxPrimitive.Trigger`, with `Button` as the rendered element. 

<details>
<summary>Suggested fix</summary>

```diff
         {showTrigger && (
-          <Button
-            type="button"
-            variant="ghost"
-            size="icon-xs"
-            render={<ComboboxTrigger />}
-            disabled={disabled}
-            data-slot="combobox-input-trigger"
-            className="group-has-data-[slot=combobox-clear]/combobox-input-group:hidden data-pressed:bg-transparent"
-          />
+          <ComboboxTrigger
+            render={<Button type="button" variant="ghost" size="icon-xs" />}
+            disabled={disabled}
+            data-slot="combobox-input-trigger"
+            className="group-has-data-[slot=combobox-clear]/combobox-input-group:hidden data-pressed:bg-transparent"
+          />
         )}
```
</details>

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
          <ComboboxTrigger
            render={<Button type="button" variant="ghost" size="icon-xs" />}
            disabled={disabled}
            data-slot="combobox-input-trigger"
            className="group-has-data-[slot=combobox-clear]/combobox-input-group:hidden data-pressed:bg-transparent"
          />
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@packages/ui/src/components/combobox.tsx` around lines 110 - 118, The Button
is receiving a non-existent render prop; move the render prop to the
ComboboxTrigger so the primitive trigger composes the Button. Remove render from
the Button instance and instead call ComboboxTrigger (or
ComboboxPrimitive.Trigger) with render set to a Button element configured with
type="button", variant="ghost", size="icon-xs", disabled={disabled},
data-slot="combobox-input-trigger" and the className currently attached; keep
any group/data attributes and ensure ComboboxTrigger receives any disabled or
aria props as needed so the Button returned by render reflects the correct
state.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Reasoning: The `Button` component invocation in `ComboboxInput` passes a `render` prop, but composition is implemented by `ComboboxPrimitive.Trigger`, not by `Button`. That means the current code is typed and wired against the wrong component boundary.
- Root cause: The trigger composition prop was attached to the rendered button instead of the combobox trigger primitive.
- Fix plan: Move `render` to `ComboboxTrigger`, keep the button as the rendered element, and preserve the disabled/data-slot/class behavior on the actual trigger wrapper.

## Resolution

- Moved the composition `render` prop onto `ComboboxTrigger` in `packages/ui/src/components/combobox.tsx` so the trigger primitive now owns the rendered button correctly.
- Added a regression in `packages/ui/src/components/combobox.test.tsx` because the existing companion test file lives outside the initial code-file list, and verified with `make verify`.
