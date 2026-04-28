---
status: resolved
file: web/src/lib/kind-colors.ts
line: 15
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM5-MtdC,comment:PRRC_kwDOR5y4QM68GE0b
---

# Issue 007: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Replace ad-hoc hex colors with design tokens.**

This map hardcodes multiple hex colors in `web/` UI code, which breaks the token-only rule and can drift from the canonical kind/status palette.

<details>
<summary>Suggested refactor</summary>

```diff
 export const KIND_COLORS: Record<string, string> = {
-  say: "#8E8E93",
-  greet: "#5BA6FF",
+  say: "var(--color-kind-say)",
+  greet: "var(--color-kind-greet)",
   direct: "var(--color-accent)",
   receipt: "var(--color-success)",
   recipe: "var(--color-warning)",
-  trace: "#B892FF",
-  whois: "#4FD1C5",
+  trace: "var(--color-kind-trace)",
+  whois: "var(--color-kind-whois)",
 };
```

</details>

As per coding guidelines: “`web/**/*.{ts,tsx,css}`: Pull every color, font, radius, spacing step, and motion value from `DESIGN.md` — never invent tokens” and “`web/**/*.{tsx,ts,css}`: Tokens live in `packages/ui/src/tokens.css`; never override with ad-hoc hex values in components”.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@web/src/lib/kind-colors.ts` around lines 7 - 15, The KIND_COLORS map
currently uses hard-coded hex values (e.g., "say": "#8E8E93", "trace":
"#B892FF") which violates the token-only rule; update the KIND_COLORS constant
in web/src/lib/kind-colors.ts to reference the canonical design tokens (CSS
variables exported from packages/ui/src/tokens.css) for every entry (replace hex
literals with var(--color-*) tokens such as var(--color-muted)/var(--color-xxx)
or the appropriate kind/status token names), ensure no ad-hoc hex remains, and
keep keys (say, greet, direct, receipt, recipe, trace, whois) unchanged so
callers still work.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause: `KIND_COLORS` hardcodes kind-dot hex values instead of referencing canonical CSS custom properties.
- Fix approach: add the missing protocol-kind color variables to `packages/ui/src/tokens.css`, surface them in the design-system showcase token wall, and update `web/src/lib/kind-colors.ts` plus the related `Pill` story to consume `var(--color-kind-*)` tokens.
