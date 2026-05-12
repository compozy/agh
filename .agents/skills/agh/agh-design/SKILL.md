---
name: agh-design
description: Use this skill to generate well-branded interfaces and assets for AGH (Compozy's open workplace for AI agents - a local-first runtime that hosts durable agent sessions and the agh-network/v0 protocol) for production UI, static HTML artifacts, slides, prototypes, mocks, and design reviews.
user-invocable: true
---

Use this skill before making AGH visual decisions. AGH is dark-mode only,
warm-dark, operator-first, and intentionally restrained.

## Triggers

- Generating UI artifacts: HTML mocks, slides, prototypes, screenshots, or production code.
- Token, color, type, spacing, radius, icon, depth, or motion decisions.
- Reviewing AI-generated UI for AGH brand and design-system alignment.

## Authority

1. `packages/ui/src/tokens.css`: canonical token source consumed by Tailwind v4.
2. `DESIGN.md`: rationale, generated token tables, anti-patterns, and semantic component contracts.
3. `packages/ui/src/components/**/*.tsx`: canonical production recipes.
4. `COPY.md`: product voice, terms, and public claim rules.

## Top-of-mind invariants

- Dark mode only; warm-dark surface ramp; one `--color-accent` target per viewport.
- Flat depth model; use `--shadow-overlay` for overlays and `--shadow-highlight` for active rims.
- Pull values from `--color-*`, `--text-*`, `--radius-*`, `--duration-*`, and `--shadow-*` tokens; do not hardcode production hex or one-off sizes.
- `<Eyebrow>` is the only uppercase label contract; do not inline typography tuples for labels.
- See `DESIGN.md` section 10 for the anti-pattern list and lint/test guardrails.

## Static HTML artifacts

Inline or import actual values from `packages/ui/src/tokens.css`. Mirror the class
structure and component anatomy in `packages/ui/src/components` where possible.
Keep artifacts dark, flat, and functional. Use literal CSS only to represent
exported token values; do not invent a parallel palette.

## Production code

Edit the owning surface: `web/`, `packages/ui/`, or `packages/site/`. Consume
CSS variables and bare Tailwind v4 token utilities. If `tokens.css` or
`packages/site/app/global.css` changes, run `make codegen` and then
`make codegen-check` so `DESIGN.md` stays synchronized.
