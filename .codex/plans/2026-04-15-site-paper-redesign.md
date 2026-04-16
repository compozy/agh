# Redesign `packages/site` Around The AGH Paper System

## Summary

- Use the Paper MCP file `AGH` as the visual source of truth, anchored on `DS Typography`, `DS Buttons`, `DS Page Layout`, `AGH Skills Page`, and `AGH Bridges — Bridge Selected`.
- Target a hybrid editorial direction: keep the dark operator grammar from the app, but translate it into a public-site experience with stronger asymmetry, better scanability, and less centered/card-grid repetition.
- Apply the redesign across both the landing page and the full docs shell, not just intro pages.
- During implementation, explicitly sequence the installed skills this way: `frontend-design` for context/anti-pattern rules, `normalize` for Paper alignment, `bolder` for landing composition, `typeset` for typography, and `polish` for the final pass. Use Paper MCP screenshots as the visual checkpoint source throughout.

## Public Interfaces / Site Contracts

- No backend API, content source, or MDX frontmatter schema changes.
- Visible site-interface changes:
  - top navigation, home header, docs shell, sidebar active states, TOC styling, page headers, and landing section composition will change
  - search result tag labeling should use `AGH Network` instead of `Protocol`
  - landing and intro docs copy may be tightened/restructured to fit the new layouts, but the approved Runtime + AGH Network positioning stays intact
- Internal contracts to update:
  - replace the default-only Fumadocs wrappers with custom home/docs shell composition rooted in `packages/site/app/*layout.tsx` and `packages/site/app/global.css`
  - expand `packages/site/mdx-components.tsx` with site-specific presentation primitives for summary rows, operator notes, comparison rows, and runtime/protocol callouts
  - remove or reconcile the stale duplicate `packages/site/lib/layout.shared.ts` so there is one navigation source of truth

## Implementation Changes

- Build the site presentation layer first:
  - preserve the existing Paper token system, dark mode, flat depth, and Inter + JetBrains Mono
  - add styles/utilities for divider-led sections, mono eyebrows, operator headers, compact pills, split layouts, and restrained surface treatments that mirror the app grammar instead of generic docs cards
  - keep no shadows, no gradients, and sparse accent usage
- Redesign the landing in `packages/site/components/landing/` around fewer, stronger compositions:
  - hero becomes left-led/asymmetric with one clear proof surface instead of a centered billboard
  - the Runtime / AGH Network split becomes editorially weighted, not two equal cards
  - getting-started becomes an operator checklist / command flow with stronger lane alignment
  - runtime capability presentation moves away from an 8-up identical grid into grouped or mixed-scale surfaces
  - AGH Network, adoption, comparison, and architecture sections should reuse Paper-style list/detail rhythm, mono labels, and divider structure rather than stacked centered blocks
  - final CTA stays runtime-first, but visually closes the page with the same operator/editorial language
- Redesign the docs shell across runtime and protocol:
  - replace the near-default Fumadocs notebook feel with a custom shell closer to the app: tighter header, clearer section grouping, subtler active row treatment, operator-like TOC rail, and stricter reading-width control
  - add a route-aware page-header treatment with eyebrow labels and stronger title/description spacing
  - keep runtime and protocol visually related, distinguishing protocol through language and supporting components rather than a second palette
  - enrich intro pages with structured MDX blocks where scanability benefits, instead of leaving them as plain text walls
- Bound content changes deliberately:
  - landing may reorder sections, merge weak repetitions, and compress copy
  - docs intro pages may be restructured to fit the new components
  - deep reference/spec pages should only be touched where the new shell exposes a real visual problem
- Implement in this order:
  1. unify navigation/layout source and theme overrides
  2. build docs shell primitives and MDX presentation components
  3. redesign landing composition and section structure
  4. adapt runtime/protocol intro pages to the new docs components
  5. run Paper-guided cleanup and final polish

## Test Plan

- Visual route review with screenshots after each major surface:
  - `/`
  - `/runtime/core/`
  - `/protocol/overview/`
  - one deep runtime reference page
  - one deep protocol/spec page
- Review every screenshot against the Paper checkpoints:
  - spacing rhythm
  - typography hierarchy
  - contrast
  - alignment and vertical lanes
  - clipping/overflow
  - repetition / AI-slop patterns
- Responsive verification:
  - desktop at approximately `1440px`
  - mobile at approximately `390px`
  - confirm nav/search/sidebar/TOC behavior still works cleanly
- Functional verification:
  - nav active states
  - search still works with updated labeling
  - sidebar groups and TOC anchors remain correct
  - primary and secondary CTAs keep their current destinations
- Repo verification before completion:
  - `bun --cwd packages/site test`
  - `bun --cwd packages/site typecheck`
  - `bun --cwd packages/site build`
  - `make verify`

## Assumptions / Defaults

- The Paper MCP file already loaded in the session is the authoritative style reference; no external export/import step is needed.
- “Hybrid editorial” means the public site gets more hierarchy, asymmetry, and breathing room than the app, but still reads unmistakably as AGH.
- The redesign stays dark and keeps Inter + JetBrains Mono; the fix is composition and shell language, not a rebrand.
- Accent orange remains a signal color, not a decorative fill spread across the whole site.
- The docs redesign is shell-wide; the landing redesign is allowed to restructure sections and compress copy, but it must preserve the existing Runtime + AGH Network positioning already approved in the repo.
