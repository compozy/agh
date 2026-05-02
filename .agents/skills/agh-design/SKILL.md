---
name: agh-design
description: Use this skill to generate well-branded interfaces and assets for AGH (Compozy's open workplace for AI agents — a local-first runtime that hosts durable agent sessions and the agh-network/v0 protocol) — either for production code or throwaway prototypes, mocks, slides, and landing pages. Contains essential design guidelines, colors, type, fonts, assets, and UI kit components for prototyping. AGH is dark-mode only with a single warm orange accent (#E8572A) on a near-black canvas (#141312); copy is operator-first, engineer-to-engineer, no emoji.
user-invocable: true
---

Read `docs/design/design-system/README.md`, and explore the other files in that folder (`colors_and_type.css`, `ui_kits/`, `preview/`, `fonts/`, and any `assets/` present).

If creating visual artifacts (slides, mocks, throwaway prototypes, etc), copy assets out and create static HTML files for the user to view. Always start from `docs/design/design-system/colors_and_type.css` so the color + type tokens match. Reuse the JSX components in `docs/design/design-system/ui_kits/marketing/` and `docs/design/design-system/ui_kits/docs/` as your building blocks.

If working on production code, you can copy assets and read the rules in `docs/design/design-system/README.md` to become an expert in designing with this brand. The real source in this repo is `packages/ui/src/tokens.css` (tokens) and `packages/site/` (marketing + docs surface).

If the user invokes this skill without any other guidance, ask them what they want to build or design (a landing section? a docs page? a slide? a product screen?), ask a few questions about scope and variations, and act as an expert designer who outputs HTML artifacts _or_ production code, depending on the need.

**Rules that matter most for this brand:**

- Dark mode only. Never render on a white background.
- One accent color: `#E8572A`. Use sparingly.
- Mono uppercase eyebrows (`JetBrains Mono`, `letter-spacing: 0.06em`) are everywhere.
- Display serif (`Playfair Display`) for marketing h1/h2; sans (`Inter`) for docs.
- No emoji. No bluish gradients. No drop shadows. No colored-left-border cards (except the one highlighted row in the comparison table).
- Copy is operator-first and dry-confident — no hype, no "we", no exclamation marks.
