---
status: resolved
file: packages/site/app/global.css
line: 255
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM57hDAH,comment:PRRC_kwDOR5y4QM64gE5F
---

# Issue 012: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

<details>
<summary>🧩 Analysis chain</summary>

🏁 Script executed:

```shell
#!/bin/bash
# Verify components that rely on ring/shadow-based styling which this reset suppresses.
rg -n --glob '*.{ts,tsx,css}' 'focus:ring|focus-visible:ring|shadow-' .
```

Repository: compozy/agh

Length of output: 17846

---



</details>

**Remove the global box-shadow reset to preserve keyboard focus indicators.**

The `box-shadow: none !important` rule suppresses focus rings across all interactive components. The codebase extensively uses Tailwind's `focus:ring-*` and `focus-visible:ring-*` utilities for keyboard focus styling (e.g., `network-protocol-visual.tsx` line 178, plus 40+ UI components), which rely on box-shadow properties. This global reset breaks keyboard navigation accessibility (WCAG 2.4.7).

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@packages/site/app/global.css` around lines 250 - 255, The global rule that
sets "box-shadow: none !important" on the universal selectors (*, *::before,
*::after) is removing Tailwind focus rings used by keyboard navigation; remove
that global box-shadow reset from global.css (or at minimum narrow its scope to
non-interactive decorative elements only) and eliminate the "!important" so
Tailwind's focus:ring-* and focus-visible:ring-* utilities (used across UI
components such as network-protocol-visual.tsx) can render focus indicators
correctly.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - The universal `box-shadow: none !important` reset suppresses Tailwind ring-based focus indicators, and this site already uses `focus:ring-*` utilities on interactive elements.
  - Root cause: the global reset applies to every element and pseudo-element, overriding the box-shadow-based focus ring implementation the components rely on.
  - Fix plan: remove the universal box-shadow reset and cover the stylesheet contract with a regression test so global CSS keeps keyboard focus styling available.
  - Resolution: removed the universal box-shadow reset from `packages/site/app/global.css` and added `packages/site/app/global.test.ts` to prevent regressions.
  - Verification: `packages/site` `bun run test`, `bun run typecheck`, and `bun run build` passed. Root `make verify` still fails outside this batch in untouched files `web/src/styles.test.ts` / `packages/ui/src/tokens.css` because the test expects `--radius: 0.5rem` while the token source defines `0.7rem`.
