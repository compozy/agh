---
status: resolved
file: packages/site/components/landing/final-cta.tsx
line: 33
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM57hDAf,comment:PRRC_kwDOR5y4QM64gE5l
---

# Issue 017: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

**Use a shared GitHub URL source to avoid drift across the site.**

Line 33 hardcodes the repo URL, while other site surfaces already carry a GitHub URL config. This can create inconsistent outbound links.

 
<details>
<summary>♻️ Suggested refactor</summary>

```diff
 import { Star } from "lucide-react";
+import { baseOptions } from "@/lib/layout.shared";
 import { CtaButton } from "./primitives/cta-button";
 import { SectionFrame } from "./primitives/section-frame";
@@
-          <a
-            href="https://github.com/compozy/agh"
+          <a
+            href={baseOptions.githubUrl}
             target="_blank"
             rel="noreferrer"
             className="mt-1 inline-flex items-center gap-2 font-mono text-[12px] uppercase tracking-(--tracking-mono) text-(--color-text-secondary) transition-colors hover:text-(--color-accent)"
           >
```
</details>

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@packages/site/components/landing/final-cta.tsx` at line 33, final-cta.tsx
currently hardcodes the GitHub href; replace that literal with the shared GitHub
URL constant used elsewhere (import the canonical symbol, e.g. GITHUB_URL or
githubUrl from the site's config/module) and use that constant in the anchor's
href in the FinalCTA component so all outbound GitHub links come from one
source; ensure you import the exact exported name used across the site and
update any imports/exports if needed so TypeScript/ESLint still pass.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - The site already has a shared GitHub URL source, but the final CTA hardcodes a different URL and the shared value itself is stale, so the drift is real.
  - Root cause: outbound GitHub links are sourced from more than one place, and `packages/site/lib/layout.shared.tsx` currently points at a fork while `FinalCta` points at the canonical org repo.
  - Fix plan: correct the shared GitHub URL in `layout.shared.tsx`, switch `FinalCta` to that shared source, and update the landing test to assert against the shared constant. This requires a minimal out-of-scope edit to `packages/site/lib/layout.shared.tsx` to eliminate the underlying drift.
  - Resolution: corrected the shared GitHub URL in `packages/site/lib/layout.shared.tsx`, switched `FinalCta` to `baseOptions.githubUrl`, and updated the landing test to assert against the shared source.
  - Verification: `packages/site` `bun run test`, `bun run typecheck`, and `bun run build` passed.
