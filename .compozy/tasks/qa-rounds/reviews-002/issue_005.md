---
status: resolved
file: web/src/systems/agent/components/agent-sessions-list.tsx
line: 124
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM59tCVn,comment:PRRC_kwDOR5y4QM67bPPW
---

# Issue 005: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Replace ad-hoc size/typography utilities with design-tokenized values.**

Line 68, Line 101, Line 107, Line 117, Line 120, and Line 123 use arbitrary values (`w-[42%]`, `text-[13px]`, `text-[10px]`, `text-[12px]`, `tracking-[0.06em]`). Please swap these to approved token-backed utilities/vars from `DESIGN.md` and `packages/ui/src/tokens.css`.  


As per coding guidelines, "Pull every color, font, radius, spacing step, and motion value from `DESIGN.md` — never invent tokens".

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@web/src/systems/agent/components/agent-sessions-list.tsx` around lines 68 -
124, The component AgentSessionRow uses ad-hoc Tailwind utilities (w-[42%],
text-[13px], text-[10px], text-[12px], tracking-[0.06em]) inside the TableHead
and Link spans; replace those arbitrary sizes with the design tokens from
DESIGN.md / packages/ui/src/tokens.css (e.g., use token-backed classes for
widths, font sizes, tracking, and text color variables) by updating the
className values in AgentSessionRow (and the TableHead element) to the
corresponding token classes/vars (font-size token for the session title,
small-caps/mono token for provider and meta text, spacing tokens for gaps, and a
width token for the Session TableHead) so all sizing/typography references come
from the approved design tokens.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - `agent-sessions-list.tsx` uses direct arbitrary Tailwind values for the session column width, item title size, provider metadata size/tracking, and metric cell size.
  - `DESIGN.md` defines the relevant typography roles (`Item Title`, `Small Body`, `Badge Text`, and mono tracking), but `packages/ui/src/tokens.css` does not currently expose Tailwind text utilities for those roles.
  - The complete fix requires a minimal token-support edit in `packages/ui/src/tokens.css` plus component updates to use token-backed utilities/vars instead of literal arbitrary values.

## Resolution

- Added Tailwind theme tokens for `text-item-title`, `text-small-body`, `text-badge`, and `tracking-mono` in `packages/ui/src/tokens.css`.
- Replaced the ad-hoc classes in `web/src/systems/agent/components/agent-sessions-list.tsx` with token-backed utilities and the standard `w-2/5` width utility.
- Verified with `make web-lint`, targeted Vitest, and full `make verify`.
