---
status: resolved
file: packages/site/components/landing/runtime-section.tsx
line: 54
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM57hDA4,comment:PRRC_kwDOR5y4QM64gE6J
---

# Issue 022: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

**`sticky` won’t take effect without an inset utility.**

On Line 54, `lg:sticky` is missing a `top-*` (or `bottom-*`) value, so the column won’t actually stick during scroll.


<details>
<summary>🔧 Suggested fix</summary>

```diff
-        <div className="h-full flex flex-col justify-between lg:sticky">
+        <div className="h-full flex flex-col justify-between lg:sticky lg:top-24">
```
</details>

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
        <div className="h-full flex flex-col justify-between lg:sticky lg:top-24">
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@packages/site/components/landing/runtime-section.tsx` at line 54, The sticky
utility on the div with className "h-full flex flex-col justify-between
lg:sticky" won't work because no inset (top/bottom) is provided; update that
element (the div in runtime-section.tsx) to include an appropriate inset utility
like lg:top-0 (or lg:top-<spacing> such as lg:top-6/lg:top-8) alongside
lg:sticky so the element actually sticks at the desired offset on large screens.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - The runtime rail currently renders `lg:sticky` with no inset utility, so the browser has no sticky offset to apply on large screens.
  - Root cause: the sticky column is missing a `top-*` value, which leaves the element behaving like regular flow content instead of pinning during scroll.
  - Fix plan: add a large-screen top offset that matches the existing layout rhythm and add a focused landing test that asserts the sticky rail includes the inset class.
  - Resolution: updated the runtime rail in `packages/site/components/landing/runtime-section.tsx` to include `lg:top-24`, which gives the sticky column an actual large-screen inset.
  - Verification: `bun run test -- components/landing/__tests__/landing.test.tsx components/logos/logos.test.tsx`, `bun run typecheck`, and `make verify` all passed.
