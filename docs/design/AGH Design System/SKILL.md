---
name: agh-design
description: Use this skill to generate well-branded interfaces and assets for AGH (Compozy's local-first agent runtime and open coordination protocol) — either for production code or throwaway prototypes, mocks, slides, and landing pages. Contains essential design guidelines, colors, type, fonts, assets, and UI kit components for prototyping. AGH is dark-mode only with a single warm orange accent (#E8572A) on a near-black canvas (#141312); copy is operator-first, engineer-to-engineer, no emoji.
user-invocable: true
---

Read the `README.md` file within this skill, and explore the other available files (`colors_and_type.css`, `ui_kits/`, `preview/`, `assets/`).

If creating visual artifacts (slides, mocks, throwaway prototypes, etc), copy assets out and create static HTML files for the user to view. Always start from `colors_and_type.css` so the color + type tokens match. Reuse the JSX components in `ui_kits/marketing/` and `ui_kits/docs/` as your building blocks.

If working on production code, you can copy assets and read the rules in `README.md` to become an expert in designing with this brand. The real source lives in `compozy/agh` @ `main`, in `packages/ui/src/tokens.css` and `packages/site/`.

If the user invokes this skill without any other guidance, ask them what they want to build or design (a landing section? a docs page? a slide? a product screen?), ask a few questions about scope and variations, and act as an expert designer who outputs HTML artifacts _or_ production code, depending on the need.

**Rules that matter most for this brand:**

- Dark mode only. Never render on a white background.
- One accent color: `#E8572A`. Use sparingly.
- Mono uppercase eyebrows (`JetBrains Mono`, `letter-spacing: 0.06em`) are everywhere.
- Display serif (`Playfair Display`) for marketing h1/h2; sans (`Inter`) for docs.
- No emoji. No bluish gradients. No drop shadows. No colored-left-border cards (except the one highlighted row in the comparison table).
- Copy is operator-first and dry-confident — no hype, no "we", no exclamation marks.
