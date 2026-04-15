---
status: resolved
file: web/src/systems/session/components/message-markdown.tsx
line: 103
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM57JF81,comment:PRRC_kwDOR5y4QM63_uII
---

# Issue 004: _⚠️ Potential issue_ | _🟠 Major_

## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Copy button is hover-only; add keyboard-visible state.**

With `opacity-0` + hover reveal only, keyboard users can tab to an effectively invisible control. Include focus-based visibility.

<details>
<summary>♿ Proposed fix</summary>

```diff
                     className={cn(
                       "absolute top-2 right-2 rounded-md p-1.5",
                       "border border-[color:var(--color-divider)] bg-[color:var(--color-surface-elevated)]",
-                      "opacity-0 transition-opacity duration-200 group-hover/codeblock:opacity-100",
+                      "opacity-0 transition-opacity duration-200",
+                      "group-hover/codeblock:opacity-100 group-focus-within/codeblock:opacity-100 focus-visible:opacity-100",
                       "text-[color:var(--color-text-tertiary)] hover:text-[color:var(--color-text-primary)]"
                     )}
```

</details>

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@web/src/systems/session/components/message-markdown.tsx` around lines 99 -
103, The copy button in MessageMarkdown is currently hidden via "opacity-0" and
only revealed on hover; make it reachable by keyboard users by adding
focus/focus-visible variants so it becomes visible when focused. Update the
class list for the copy button element (the JSX that currently includes
"opacity-0 transition-opacity duration-200 group-hover/codeblock:opacity-100")
to also include "focus:opacity-100 focus-visible:opacity-100" (or the equivalent
"group-focus/codeblock:opacity-100" if using a group focus approach) so tabbing
to the button shows it.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause: The code-block copy control starts at `opacity-0` and only gains visibility through `group-hover/codeblock:opacity-100`, so keyboard users can tab onto an effectively invisible button with no focus-triggered reveal.
- Fix approach: Add focus-driven visibility classes on the existing `CopyButton` instance and add a focused regression test for the code-block copy button because no dedicated `message-markdown` test currently covers this accessibility path.
- Resolution: Added `group-focus-within/codeblock:opacity-100` and `focus-visible:opacity-100` to the code-block copy button, plus a new `message-markdown.test.tsx` regression test covering the keyboard-visible class contract. Verified with focused tests, `make web-lint`, `make web-typecheck`, and `make verify`.
