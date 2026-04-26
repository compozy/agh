---
status: resolved
file: web/src/styles.css
line: 20
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM59oaQ8,comment:PRRC_kwDOR5y4QM67VX7b
---

# Issue 022: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Use design-system motion tokens instead of hardcoded transition values.**

Line 18 and Line 19 hardcode `140ms` and `ease-out`. Please source these from `DESIGN.md`/tokenized CSS variables so motion stays consistent across the app.

As per coding guidelines, `web/src/**/*.{tsx,ts,css}`: Pull every color, font, radius, spacing step, and motion value from `DESIGN.md` in the repo root — never invent tokens.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@web/src/styles.css` around lines 16 - 20, The view-transition pseudo-elements
(::view-transition-old(root) and ::view-transition-new(root)) currently hardcode
animation-duration: 140ms and animation-timing-function: ease-out; replace those
literals with the design-system motion tokens defined in DESIGN.md (use the
repo's CSS token variables for duration and easing, e.g. the appropriate
--motion-duration-... and --motion-ease-... tokens) by updating the
animation-duration and animation-timing-function properties to reference those
CSS variables so motion values are centralized and consistent across the app.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - `web/src/styles.css` hardcodes `140ms` and `ease-out` for root view transitions even though the design system already defines shared duration/easing tokens.
  - Root cause: the stylesheet bypasses `packages/ui/src/tokens.css` for motion values, creating an app-local divergence from `DESIGN.md`.
  - Fix plan: switch the transition declarations to the shared CSS variables and add a small source regression in `web/src/storybook/web-storybook-stories-and-fixtures.test.tsx` because no scoped runtime test currently covers stylesheet token usage.
