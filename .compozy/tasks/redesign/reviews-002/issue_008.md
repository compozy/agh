---
status: resolved
file: packages/ui/src/components/chat-message-bubble.tsx
line: 33
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM57_lcG,comment:PRRC_kwDOR5y4QM65JoyG
---

# Issue 008: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**`align` is public but not visually honored for non-`user` roles**

Line 15 exposes `align` generally, but in Lines 104 and 130 the class composition ignores `resolvedAlign`. For `agent`/`tool`/`diff`, `align="right"` updates `data-align` only, not layout. This is an API behavior mismatch.


<details>
<summary>🛠️ Suggested fix (honor alignment consistently)</summary>

```diff
   if (role === "agent") {
     return (
       <div
@@
-        className={cn("flex w-full flex-col gap-1.5", className)}
+        className={cn(
+          "flex w-full flex-col gap-1.5",
+          resolvedAlign === "right" ? "items-end text-right" : "items-start text-left",
+          className
+        )}
         {...props}
       >
@@
   return (
     <div
@@
-      className={cn("flex w-full flex-col gap-1.5", className)}
+      className={cn(
+        "flex w-full flex-col gap-1.5",
+        resolvedAlign === "right" ? "items-end text-right" : "items-start text-left",
+        className
+      )}
       {...props}
     >
```
</details>



Also applies to: 98-123, 125-143

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@packages/ui/src/components/chat-message-bubble.tsx` at line 33, The component
currently computes resolvedAlign (const resolvedAlign: ChatMessageAlign = align
?? (role === "user" ? "right" : "left")) but only applies it to data-align in
some branches, so passing align="right" for non-user roles doesn't change the
layout; update all className compositions that control layout (the branches
rendering the message container, avatar/message order, and bubble alignment) to
use resolvedAlign rather than role-based logic so layout classes (e.g., flex
direction, margin-start/margin-end, text alignment) reflect resolvedAlign
consistently while still setting data-align for styling hooks; search for usages
around the message container render (where data-align is currently set) and
replace the role-derived class decisions with checks against resolvedAlign so
agent/tool/diff messages honor align prop.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Reasoning: The component computes `resolvedAlign` for every role, but the `agent`, `tool`, and `diff` branches still hard-code left-aligned layout classes. Passing `align="right"` changes only `data-align`, not the rendered layout.
- Root cause: Layout classes in the non-user branches are driven by role-specific defaults instead of the resolved public `align` prop.
- Fix plan: Apply `resolvedAlign` consistently to the relevant layout and text-alignment classes, then cover the behavior with tests.

## Resolution

- Updated `packages/ui/src/components/chat-message-bubble.tsx` so `agent`, `tool`, and `diff` branches use `resolvedAlign` for layout/meta alignment, not just `data-align`.
- Covered the behavior in `packages/ui/src/components/chat-message-bubble.test.tsx` and verified with `make verify`.
