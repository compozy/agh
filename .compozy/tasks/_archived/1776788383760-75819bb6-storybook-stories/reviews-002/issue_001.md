---
status: resolved
file: web/src/components/ui/stories/collapsible.stories.tsx
line: 83
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM575wGt,comment:PRRC_kwDOR5y4QM65CLBx
---

# Issue 001: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

<details>
<summary>🧩 Analysis chain</summary>

🏁 Script executed:

```shell
#!/bin/bash
# Verify where open-state selectors are applied in the trigger vs icon.
rg -n -C2 'group-data-\[panel-open\]|data-\[panel-open\]|CollapsibleTrigger|CollapsiblePrimitive\.Trigger' \
  web/src/components/ui/stories/collapsible.stories.tsx \
  web/src/components/ui/collapsible.tsx
```

Repository: compozy/agh

Length of output: 3577

---



</details>

**Icon rotation state selector is incorrect.**

On Line 80, `data-[panel-open]:rotate-180` is applied directly to `ChevronDownIcon`, but the `panel-open` state is emitted by the `CollapsibleTrigger` element, not the icon. This means the rotation won't toggle as intended.

The first example in the same file (lines 42-49) demonstrates the correct pattern—use `group-data-[panel-open]/collapsible-trigger:` selectors on child elements to reference the trigger's state.

<details>
<summary>🔧 Suggested fix</summary>

```diff
         <CollapsibleTrigger
+          className="group/collapsible-trigger"
           render={
             <Button variant="ghost" size="sm" className="gap-1.5">
               Hide timeline
-              <ChevronDownIcon className="size-4 transition-transform data-[panel-open]:rotate-180" />
+              <ChevronDownIcon className="size-4 transition-transform group-data-[panel-open]/collapsible-trigger:rotate-180" />
             </Button>
           }
         />
```
</details>

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
        <Collapsible defaultOpen>
          <CollapsibleTrigger
            className="group/collapsible-trigger"
            render={
              <Button variant="ghost" size="sm" className="gap-1.5">
                Hide timeline
                <ChevronDownIcon className="size-4 transition-transform group-data-[panel-open]/collapsible-trigger:rotate-180" />
              </Button>
            }
          />
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@web/src/components/ui/stories/collapsible.stories.tsx` around lines 75 - 83,
The ChevronDownIcon rotation is using data-[panel-open] directly but that state
is emitted by CollapsibleTrigger, so update the trigger to be a group and apply
the group-data selector on the icon: add a group class to the
CollapsibleTrigger's render Button (e.g., className includes "group" or
"group/collapsible-trigger" consistent with your selectors) and replace the
icon's "data-[panel-open]:rotate-180" with the corresponding group selector
"group-data-[panel-open]/collapsible-trigger:rotate-180" (modify the className
string on ChevronDownIcon and the Button inside CollapsibleTrigger) so the icon
rotates when CollapsibleTrigger toggles.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - Verified against `web/src/components/ui/stories/collapsible.stories.tsx` and `web/src/components/ui/collapsible.tsx`.
  - `CollapsibleTrigger` emits the `panel-open` state, and the `OpenByDefault` story currently applies `data-[panel-open]:rotate-180` directly on `ChevronDownIcon`, which never receives that state attribute.
  - Root cause: the `OpenByDefault` story is missing the trigger group class and uses the wrong selector for the icon state.
  - Fix approach: mirror the working `Default` story pattern by making the trigger a named group and switching the icon to the corresponding `group-data-[panel-open]/...` selector.
  - Resolved by updating the story selector and covering it in `web/src/storybook/web-storybook-stories-and-fixtures.test.tsx`.
