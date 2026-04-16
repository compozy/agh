---
status: resolved
file: packages/site/components/logos/linear.tsx
line: 239
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM57hDBM,comment:PRRC_kwDOR5y4QM64gE6m
---

# Issue 025: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Prevent SVG definition ID collisions across multiple renders.**

The icon variant uses fixed IDs (`a`, `b`, `c`, ...). Rendering multiple `LinearLogo` instances in one DOM can break gradients/filters due to global ID collisions.

<details>
<summary>Suggested patch pattern</summary>

```diff
+import { useId } from "react";
...
 export function LinearLogo({ className, variant = "logo", mode = "dark" }: LinearLogoProps) {
   const color = COLORS[mode];
+  const uid = useId().replace(/:/g, "");
+  const ids = {
+    a: `linear-${uid}-a`,
+    b: `linear-${uid}-b`,
+    // ...repeat for all defs/refs
+  };

   if (variant === "icon") {
     return (
       <svg ...>
-        <path fill="url(`#a`)" d="M0 0h512v512H0z" />
-        <g filter="url(`#b`)" opacity=".8">
+        <path fill={`url(#${ids.a})`} d="M0 0h512v512H0z" />
+        <g filter={`url(#${ids.b})`} opacity=".8">
...
-          <linearGradient id="a" ...>
+          <linearGradient id={ids.a} ...>
...
-          <filter id="b" ...>
+          <filter id={ids.b} ...>
```
</details>

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@packages/site/components/logos/linear.tsx` around lines 28 - 239, The SVG
uses fixed IDs (a, b, c, ... and filter refs like url(`#b`)) in the "icon" branch
which causes collisions when multiple LinearLogo components render; update the
component (LinearLogo or the function that returns the SVG for variant ===
"icon") to generate a stable unique prefix per instance (e.g., React's useId()
or an idPrefix prop) and prepend it to every id and every reference (e.g.,
id={`${prefix}-a`} and fill={`url(#${prefix}-a)`}, filter={`url(#${prefix}-b)`},
etc.) for all gradient/filter/radial/filter ids (a..n and b,c,e) so each SVG's
defs and references remain isolated.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - The `LinearLogo` icon variant hardcodes gradient, radial-gradient, and filter IDs (`a` through `n`) and reuses those same fragment references for every render.
  - Root cause: the icon branch does not namespace its SVG defs per component instance, so multiple icons can collide in the shared DOM ID space.
  - Fix plan: derive a sanitized per-instance prefix with `useId()`, prepend it to all icon-branch defs and `url(#...)` references, and add a regression test that renders two icon instances and verifies all generated IDs are unique.
  - Resolution: updated the `LinearLogo` icon branch to derive every gradient/filter ID and `url(#...)` reference from a sanitized per-instance `useId()` prefix.
  - Verification: `bun run test -- components/landing/__tests__/landing.test.tsx components/logos/logos.test.tsx`, `bun run typecheck`, and `make verify` all passed.
