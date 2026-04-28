---
status: resolved
file: packages/ui/src/components/pill-group.tsx
line: 19
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM5-Mtcr,comment:PRRC_kwDOR5y4QM68GEz-
---

# Issue 001: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Replace hard-coded pixel classes with DESIGN.md tokens.**

Line 9, Line 17, Line 18, Line 62, Line 89, and nearby classes introduce raw px values (`[5px]`, `[10px]`, `[20px]`, `[22px]`, `[2px]`, `[3px]`, `[14px]`, `[7px]`, `[9px]`). In `packages/ui/src`, these should come from design tokens only to keep the system consistent.

  
As per coding guidelines, "`**/{web,packages/ui,packages/site}/src/**/*.{ts,tsx,css}`: Pull design tokens from `DESIGN.md` ... never invent values in UI code."


Also applies to: 62-63, 89-90

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@packages/ui/src/components/pill-group.tsx` around lines 9 - 19, The PillGroup
component contains hard-coded pixel classes (e.g., rounded-[5px], text-[10px],
h-[20px], h-[22px], px-2 (if representing 8px/10px), and other
[14px],[7px],[9px] usages) which must be replaced with the design token
equivalents from DESIGN.md; update the class strings inside
packages/ui/src/components/pill-group.tsx (the variant block for active and
size, base class string, and any other nearby class values referenced in this
file) to use the canonical DESIGN.md tokens (CSS custom properties or token
utility classes used across the repo) instead of raw px values, mapping each
specific token (rounded, font-size, height, padding, gap) to the appropriate
DESIGN.md name so no new numeric values are invented and all sizes reference the
shared tokens.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause: `PillGroup` still encodes badge/segment dimensions with raw arbitrary pixel classes (`rounded-[5px]`, `h-[22px]`, `p-[3px]`, `text-[9px]`, etc.) instead of design-system tokens or token-derived utilities.
- Fix approach: replace raw px classes with existing design tokens where available (`--radius-chip`, `--radius`, `text-badge`, spacing scale) and add minimal shared component sizing custom properties in `packages/ui/src/tokens.css` where DESIGN.md defines values that do not already have a token.
