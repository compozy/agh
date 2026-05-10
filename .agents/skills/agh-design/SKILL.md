---
name: agh-design
description: Use this skill to generate well-branded interfaces and assets for AGH (Compozy's open workplace for AI agents — a local-first runtime that hosts durable agent sessions and the agh-network/v0 protocol) — either for production code or throwaway prototypes, mocks, slides, and landing pages. Contains essential design guidelines, colors, type, fonts, assets, and UI kit components for prototyping. AGH is dark-mode only with a single warm orange accent (#E8572A) on a near-black canvas (#131211); copy is operator-first, engineer-to-engineer, no emoji.
user-invocable: true
---

The authoritative design system lives in two places — read both before producing anything:

1. `DESIGN.md` (repo root) — the canonical specification: visual theme, color tokens, surface ramp, hairlines, type, motion, ADRs, anti-patterns.
2. `packages/ui/src/tokens.css` — the canonical CSS token source consumed by the runtime UI, the marketing site, and the docs site.

Supporting sources to pull from when building:

- `packages/ui/src/components/` — shadcn-derived primitives (`button.tsx`, `card.tsx`, `dialog.tsx`, `popover.tsx`, `sidebar.tsx`, `tabs.tsx`, `table.tsx`, `select.tsx`, `field.tsx`, `tooltip.tsx`, `sonner.tsx`, etc.) plus AGH-custom variants under `packages/ui/src/components/custom/`.
- `web/` — runtime operator UI (React 19 + TanStack + Tailwind v4); use it to see how tokens compose in product surfaces.
- `packages/site/` — Fumadocs marketing landing + docs at `agh.network`; use it to see the editorial Playfair Display rhythm and `/runtime/*` + `/protocol/*` doc trees.

If creating visual artifacts (slides, mocks, throwaway prototypes, landing sections), produce static HTML files the user can open directly. Pull the actual tokens by importing or inlining values from `packages/ui/src/tokens.css` — never invent hex values. Reuse the JSX components from `packages/ui/src/components/` as building blocks when the artifact is React/TSX; for pure HTML, mirror their class structure and token usage.

If working on production code, edit inside `web/`, `packages/ui/`, or `packages/site/` and consume tokens via the existing CSS variables (`var(--canvas)`, `var(--accent)`, `var(--line)`, etc.) — never hardcode hex.

If invoked without other guidance, ask what the user wants to build (a landing section? a docs page? a slide? a product screen?), gather scope and variations, and act as an expert designer outputting HTML artifacts _or_ production code depending on the need.

**Rules that matter most for this brand (full reasoning in `DESIGN.md`):**

- **Dark mode only.** `color-scheme: dark` is hardcoded. Never render on a white background.
- **One accent: `#E8572A`** (`--accent`). Use sparingly — one accent target per viewport.
- **Warm-tinted neutral ramp**, never cool/bluish. Surface ramp: `--rail` `#0c0b0b` → `--canvas` `#131211` → `--canvas-soft` `#1a1918` → `--canvas-tint` `#1c1b1a` → `--elevated` `#232220`.
- **Translucent hairlines, not solid borders.** Use `--line`, `--line-soft`, `--line-strong` (rgba whites). The legacy solid `#3C3A39` divider is gone.
- **Two-token shadow vocabulary (ADR-003).** Only `--shadow-overlay` (modals) and `--highlight` (active rim) carry shadow. Cards, popovers, dropdowns, sticky headers, list rows stay flat with a 1 px ring on `--line-soft`. No drop shadows on content, no glassmorphism, no decorative texture.
- **Type stack:** Inter Variable for UI, JetBrains Mono for metadata + mono uppercase eyebrows (`letter-spacing: 0.06em`). Playfair Display reserved for `packages/site/` marketing hero only. NuixyberNext reserved for the `agh` wordmark only.
- **Signal colors are desaturated and tint-only.** Success `#5fbf85`, warning `#d6a647`, danger `#e0635a`, info `#8e8eb5`. Appear as low-alpha tints on chips/pills, never as solid banners.
- **No emoji. No bluish gradients. No colored-left-border cards** (except the one highlighted row in the comparison table).
- **Copy is operator-first and dry-confident** — engineer-to-engineer, no hype, no "we", no exclamation marks. See `COPY.md` for the full product-language spec.
