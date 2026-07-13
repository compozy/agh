---
name: agh-design
description: AGH visual-design authority for production UI, static artifacts, prototypes, and reviews. Use when creating or reviewing an AGH surface or changing tokens, typography, spacing, depth, icons, or motion. Do not use for capture-only verification; use agh-ui-screenshot.
---

# AGH Design

Use the canonical visual authorities before making AGH design decisions.

## Authority

1. `packages/ui/src/tokens.css`: canonical token source consumed by Tailwind v4.
2. `DESIGN.md`: rationale, generated token tables, anti-patterns, and semantic component contracts.
3. `packages/ui/src/index.ts`: the `@agh/ui` surface contract — the canonical primitive inventory.
4. `packages/ui/src/components/**/*.tsx`: canonical production recipes.
5. `COPY.md`: product voice, terms, and public claim rules.

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

Reuse gate — before authoring any component, map every generic UI need against
`packages/ui/src/index.ts` and import the primitive from `@agh/ui` instead of
re-implementing it. Redefining an exported name in `web/` or `packages/site/`
fails lint (`compozy-ui-reuse/no-shadow-ui-primitive`); genuinely
domain-specific variants take a domain-prefixed name. New generic primitives
land in `packages/ui` with story + test; domain composites in
`web/src/systems/<domain>/`.

Edit the owning surface: `web/`, `packages/ui/`, or `packages/site/`. Consume
CSS variables and bare Tailwind v4 token utilities. If `tokens.css` or
`packages/site/app/global.css` changes, run `make codegen` and then
`make codegen-check` so `DESIGN.md` stays synchronized.

## Completion Criterion

The change is complete when it uses the canonical tokens and primitive owner,
introduces no parallel palette or shadow primitive, respects the owning
surface's instructions, and regenerated design artifacts have no drift when a
token source changed.

## Error Handling

- **`DESIGN.md` and runtime tokens disagree:** treat `packages/ui/src/tokens.css` as source, run codegen, and inspect the regenerated spec rather than editing generated regions.
- **No exported primitive fits:** decide whether the need is generic or domain-specific; add generic primitives to `packages/ui` and domain composites to the owning Web system.
- **A plausible mock implies unsupported runtime behavior:** remove the unsupported control or metric; daemon truth wins.
