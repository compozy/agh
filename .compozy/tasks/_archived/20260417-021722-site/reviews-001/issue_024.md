---
status: resolved
file: packages/site/components/logos/gemini.tsx
line: 145
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM57hDBB,comment:PRRC_kwDOR5y4QM64gE6V
---

# Issue 024: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

**Avoid static SVG IDs to prevent cross-instance rendering collisions.**

Line 18 and the filter IDs at Lines 56/69/82/95/108/121/134 are static. If `GeminiLogo` is rendered more than once on a page, masks/filters can bind to the wrong element.


<details>
<summary>💡 Suggested fix</summary>

```diff
 import { cn } from "@agh/ui/utils";
+import { useId } from "react";

 export interface GeminiLogoProps extends React.SVGProps<SVGSVGElement> {}

 export function GeminiLogo({ className, ...props }: GeminiLogoProps) {
+  const uid = useId();
+  const maskA = `${uid}-gemini-a`;
+  const filterB = `${uid}-gemini-b`;
+  const filterC = `${uid}-gemini-c`;
+  const filterD = `${uid}-gemini-d`;
+  const filterE = `${uid}-gemini-e`;
+  const filterF = `${uid}-gemini-f`;
+  const filterG = `${uid}-gemini-g`;
+  const filterH = `${uid}-gemini-h`;
+
   return (
     <svg
@@
-      <mask
-        id="gemini__a"
+      <mask
+        id={maskA}
@@
-      <g mask="url(`#gemini__a`)">
-        <g filter="url(`#gemini__b`)">
+      <g mask={`url(#${maskA})`}>
+        <g filter={`url(#${filterB})`}>
@@
-        <g filter="url(`#gemini__c`)">
+        <g filter={`url(#${filterC})`}>
@@
-        <g filter="url(`#gemini__d`)">
+        <g filter={`url(#${filterD})`}>
@@
-        <g filter="url(`#gemini__e`)">
+        <g filter={`url(#${filterE})`}>
@@
-        <g filter="url(`#gemini__f`)">
+        <g filter={`url(#${filterF})`}>
@@
-        <g filter="url(`#gemini__g`)">
+        <g filter={`url(#${filterG})`}>
@@
-        <g filter="url(`#gemini__h`)">
+        <g filter={`url(#${filterH})`}>
@@
-        <filter id="gemini__b"
+        <filter id={filterB}
@@
-        <filter id="gemini__c"
+        <filter id={filterC}
@@
-        <filter id="gemini__d"
+        <filter id={filterD}
@@
-        <filter id="gemini__e"
+        <filter id={filterE}
@@
-        <filter id="gemini__f"
+        <filter id={filterF}
@@
-        <filter id="gemini__g"
+        <filter id={filterG}
@@
-        <filter id="gemini__h"
+        <filter id={filterH}
```
</details>

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
import { cn } from "@agh/ui/utils";
import { useId } from "react";

export interface GeminiLogoProps extends React.SVGProps<SVGSVGElement> {}

export function GeminiLogo({ className, ...props }: GeminiLogoProps) {
  const uid = useId();
  const maskA = `${uid}-gemini-a`;
  const filterB = `${uid}-gemini-b`;
  const filterC = `${uid}-gemini-c`;
  const filterD = `${uid}-gemini-d`;
  const filterE = `${uid}-gemini-e`;
  const filterF = `${uid}-gemini-f`;
  const filterG = `${uid}-gemini-g`;
  const filterH = `${uid}-gemini-h`;

  return (
    <svg
      className={cn("w-10 h-10", className)}
      fill="none"
      viewBox="0 0 296 298"
      xmlns="http://www.w3.org/2000/svg"
      {...props}
    >
      <mask
        id={maskA}
        width="296"
        height="298"
        x="0"
        y="0"
        maskUnits="userSpaceOnUse"
        style={{ maskType: "alpha" }}
      >
        <path
          fill="#3186FF"
          d="M141.201 4.886c2.282-6.17 11.042-6.071 13.184.148l5.985 17.37a184.004 184.004 0 0 0 111.257 113.049l19.304 6.997c6.143 2.227 6.156 10.91.02 13.155l-19.35 7.082a184.001 184.001 0 0 0-109.495 109.385l-7.573 20.629c-2.241 6.105-10.869 6.121-13.133.025l-7.908-21.296a184 184 0 0 0-109.02-108.658l-19.698-7.239c-6.102-2.243-6.118-10.867-.025-13.132l20.083-7.467A183.998 183.998 0 0 0 133.291 26.28l7.91-21.394Z"
        />
      </mask>
      <g mask={`url(#${maskA})`}>
        <g filter={`url(#${filterB})`}>
          <ellipse cx="163" cy="149" fill="#3689FF" rx="196" ry="159" />
        </g>
        <g filter={`url(#${filterC})`}>
          <ellipse cx="33.5" cy="142.5" fill="#F6C013" rx="68.5" ry="72.5" />
        </g>
        <g filter={`url(#${filterD})`}>
          <ellipse cx="19.5" cy="148.5" fill="#F6C013" rx="68.5" ry="72.5" />
        </g>
        <g filter={`url(#${filterE})`}>
          <path fill="#FA4340" d="M194 10.5C172 82.5 65.5 134.333 22.5 135L144-66l50 76.5Z" />
        </g>
        <g filter={`url(#${filterF})`}>
          <path fill="#FA4340" d="M190.5-12.5C168.5 59.5 62 111.333 19 112L140.5-89l50 76.5Z" />
        </g>
        <g filter={`url(#${filterG})`}>
          <path fill="#14BB69" d="M194.5 279.5C172.5 207.5 66 155.667 23 155l121.5 201 50-76.5Z" />
        </g>
        <g filter={`url(#${filterH})`}>
          <path fill="#14BB69" d="M196.5 320.5C174.5 248.5 68 196.667 25 196l121.5 201 50-76.5Z" />
        </g>
      </g>
      <defs>
        <filter
          id={filterB}
          width="464"
          height="390"
          x="-69"
          y="-46"
          colorInterpolationFilters="sRGB"
          filterUnits="userSpaceOnUse"
        >
          <feFlood floodOpacity="0" result="BackgroundImageFix" />
          <feBlend in="SourceGraphic" in2="BackgroundImageFix" result="shape" />
          <feGaussianBlur result="effect1_foregroundBlur_69_17998" stdDeviation="18" />
        </filter>
        <filter
          id={filterC}
          width="265"
          height="273"
          x="-99"
          y="6"
          colorInterpolationFilters="sRGB"
          filterUnits="userSpaceOnUse"
        >
          <feFlood floodOpacity="0" result="BackgroundImageFix" />
          <feBlend in="SourceGraphic" in2="BackgroundImageFix" result="shape" />
          <feGaussianBlur result="effect1_foregroundBlur_69_17998" stdDeviation="32" />
        </filter>
        <filter
          id={filterD}
          width="265"
          height="273"
          x="-113"
          y="12"
          colorInterpolationFilters="sRGB"
          filterUnits="userSpaceOnUse"
        >
          <feFlood floodOpacity="0" result="BackgroundImageFix" />
          <feBlend in="SourceGraphic" in2="BackgroundImageFix" result="shape" />
          <feGaussianBlur result="effect1_foregroundBlur_69_17998" stdDeviation="32" />
        </filter>
        <filter
          id={filterE}
          width="299.5"
          height="329"
          x="-41.5"
          y="-130"
          colorInterpolationFilters="sRGB"
          filterUnits="userSpaceOnUse"
        >
          <feFlood floodOpacity="0" result="BackgroundImageFix" />
          <feBlend in="SourceGraphic" in2="BackgroundImageFix" result="shape" />
          <feGaussianBlur result="effect1_foregroundBlur_69_17998" stdDeviation="32" />
        </filter>
        <filter
          id={filterF}
          width="299.5"
          height="329"
          x="-45"
          y="-153"
          colorInterpolationFilters="sRGB"
          filterUnits="userSpaceOnUse"
        >
          <feFlood floodOpacity="0" result="BackgroundImageFix" />
          <feBlend in="SourceGraphic" in2="BackgroundImageFix" result="shape" />
          <feGaussianBlur result="effect1_foregroundBlur_69_17998" stdDeviation="32" />
        </filter>
        <filter
          id={filterG}
          width="299.5"
          height="329"
          x="-41"
          y="91"
          colorInterpolationFilters="sRGB"
          filterUnits="userSpaceOnUse"
        >
          <feFlood floodOpacity="0" result="BackgroundImageFix" />
          <feBlend in="SourceGraphic" in2="BackgroundImageFix" result="shape" />
          <feGaussianBlur result="effect1_foregroundBlur_69_17998" stdDeviation="32" />
        </filter>
        <filter
          id={filterH}
          width="299.5"
          height="329"
          x="-39"
          y="132"
          colorInterpolationFilters="sRGB"
          filterUnits="userSpaceOnUse"
        >
          <feFlood floodOpacity="0" result="BackgroundImageFix" />
          <feBlend in="SourceGraphic" in2="BackgroundImageFix" result="shape" />
          <feGaussianBlur result="effect1_foregroundBlur_69_17998" stdDeviation="32" />
        </filter>
      </defs>
    </svg>
  );
}
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@packages/site/components/logos/gemini.tsx` around lines 17 - 145, The SVG
uses static IDs (mask "gemini__a" and filters "gemini__b"..."gemini__h") which
will collide when GeminiLogo is rendered multiple times; update the component
(GeminiLogo) to generate a per-instance unique suffix (e.g., React's useId() or
a stable random/uuid) and append it to every ID and all corresponding references
(mask="url(#...)" and filter="url(#...)") so each instance uses unique IDs like
`${base}__a_${uid}` for the mask and `${base}__b_${uid}` etc.; ensure the same
uid is applied consistently across <mask>, <defs> ids and every mask="url(#...)"
/ filter="url(#...)" attribute so references still match.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - `GeminiLogo` hardcodes one mask ID and seven filter IDs, and every `mask="url(#...)"` / `filter="url(#...)"` reference points to those fixed DOM IDs.
  - Root cause: the SVG definition IDs are static, so rendering multiple `GeminiLogo` instances on one page can cross-wire defs and references between instances.
  - Fix plan: generate a sanitized per-instance prefix with `useId()`, rewrite every Gemini def/reference to use that prefix, and add a multi-render regression test that proves the generated IDs do not collide.
  - Resolution: rewired every Gemini mask/filter definition and reference to use a sanitized per-instance `useId()` prefix in `packages/site/components/logos/gemini.tsx`.
  - Verification: `bun run test -- components/landing/__tests__/landing.test.tsx components/logos/logos.test.tsx`, `bun run typecheck`, and `make verify` all passed.
