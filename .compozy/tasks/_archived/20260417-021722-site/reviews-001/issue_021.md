---
status: resolved
file: packages/site/components/landing/runtime-micro-diagram.tsx
line: 141
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM57hDAx,comment:PRRC_kwDOR5y4QM64gE5-
---

# Issue 021: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

**Guard animation with `prefers-reduced-motion` at CSS level.**

Line 123 conditionally injects animation styles based on runtime state; because `useReducedMotion()` starts as `false`, reduced-motion users can still see a brief animation before hydration settles. Add a media-query guard so animations are suppressed immediately by the browser preference.



<details>
<summary>Suggested patch</summary>

```diff
       {!reducedMotion && (
         <style>{`
           .agh-subsystem {
             animation-name: agh-subsystem-pulse;
             animation-iteration-count: infinite;
             animation-timing-function: ease-in-out;
           }
+          `@media` (prefers-reduced-motion: reduce) {
+            .agh-subsystem {
+              animation: none !important;
+            }
+          }
           `@keyframes` agh-subsystem-pulse {
             0%, 100% {
               fill: var(--color-surface);
               stroke: var(--color-divider);
             }
```
</details>

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
      {!reducedMotion && (
        <style>{`
          .agh-subsystem {
            animation-name: agh-subsystem-pulse;
            animation-iteration-count: infinite;
            animation-timing-function: ease-in-out;
          }
          `@media` (prefers-reduced-motion: reduce) {
            .agh-subsystem {
              animation: none !important;
            }
          }
          `@keyframes` agh-subsystem-pulse {
            0%, 100% {
              fill: var(--color-surface);
              stroke: var(--color-divider);
            }
            10%, 30% {
              fill: var(--color-accent-tint);
              stroke: var(--color-accent);
            }
          }
        `}</style>
      )}
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@packages/site/components/landing/runtime-micro-diagram.tsx` around lines 123
- 141, The injected animation CSS is only conditional on the runtime
reducedMotion prop (reducedMotion) which can cause a flash before hydration; add
a CSS-level guard using the prefers-reduced-motion media query inside the style
block so the browser immediately disables animations for users who prefer
reduced motion. Specifically, update the style injected by the component
rendering .agh-subsystem and `@keyframes` (in the same template string around
where reducedMotion is checked) to include a top-level `@media`
(prefers-reduced-motion: reduce) { .agh-subsystem { animation: none !important;
} } (and if desired also override animation-iteration-count/animation-name) to
ensure no animation runs before React hydration.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - `RuntimeMicroDiagram` is a client component and `useReducedMotion()` starts as `false`, so reduced-motion users can receive the animation class and keyframes in the initial HTML before hydration corrects the state.
  - Root cause: the injected style block only gates animation at React render time; it has no CSS-level `prefers-reduced-motion` override that the browser can apply immediately.
  - Fix plan: add an `@media (prefers-reduced-motion: reduce)` override inside the injected style block and add a regression test that asserts the media-query guard is present.
  - Resolution: added the `prefers-reduced-motion` media-query override in `packages/site/components/landing/runtime-micro-diagram.tsx` so the browser suppresses the animation before hydration settles.
  - Verification: `bun run test -- components/landing/__tests__/landing.test.tsx components/logos/logos.test.tsx`, `bun run typecheck`, and `make verify` all passed.
