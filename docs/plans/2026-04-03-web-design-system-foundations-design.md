# Web Design System Foundations

## Context

AGH is an agent operating system for developers and operators managing live agent sessions. The current `web/` package already has a broad raw shadcn/base-ui inventory, but it does not yet express a product-specific visual language.

This pass establishes the foundation layer only. It does not attempt to restyle the entire primitive library.

## Audience And Tone

- Audience: developers and operators supervising agent sessions, runtime state, and system workflows
- Primary use case: fast scanning of live information, compact decision-making, and reliable control surfaces
- Tone: dark, precise, high-signal, and intentionally dense

The visual references in `docs/design/` point toward a graphite control-room aesthetic with subtle texture, warm action accents, compact pills, mono metadata, and rounded matte surfaces.

## Goals

- Introduce project-native visual tokens in `web/src/styles.css`
- Build a small reusable foundation component layer on top of the raw `ui/` primitives
- Replace the placeholder home route with a living showcase for the new system
- Keep the migration surface narrow so later screens can adopt the foundations incrementally

## Token Taxonomy

### Canvas

- `--ds-canvas`
- `--ds-canvas-strong`
- `--ds-canvas-glow`
- `--ds-vignette`
- `--ds-texture-line`

### Surface

- `--ds-panel-base`
- `--ds-panel-elevated`
- `--ds-panel-accent`
- `--ds-line-subtle`
- `--ds-line-strong`
- `--ds-inner-highlight`

### Text

- `--ds-text-primary`
- `--ds-text-secondary`
- `--ds-text-muted`
- `--ds-text-mono`

### Accent

- `--ds-accent-amber`
- `--ds-accent-green`
- `--ds-accent-violet`
- `--ds-accent-danger`

### Shape And Depth

- `--ds-radius-shell`
- `--ds-radius-panel`
- `--ds-radius-pill`
- `--ds-shadow-panel`
- `--ds-shadow-focus`

## Primitive Set

The first-pass shared primitives are:

- `texture-canvas`
- `app-shell`
- `panel`
- `section-heading`
- `pill`
- `status-dot`
- `metric-strip`
- `toolbar`

These primitives sit beside the raw `ui/` inventory. They are the preferred building blocks for new product-facing surfaces.

## Showcase Composition

The home route becomes a command-surface preview:

- hero heading with product context and design-system framing
- toolbar with compact pills, search shell, and accent action
- asymmetrical panel grid showing mission threads, system metrics, and integration health
- token and primitive preview section so the route doubles as a living system reference

The page should feel inspired by the references without cloning any single screen literally.

## Migration Rules

- Keep `web/src/components/ui/*` intact in this pass
- Do not broad-restyle the shadcn inventory yet
- Migrate only the home route to use the new foundation layer
- Future routes should prefer the foundation components first and adopt raw `ui/` primitives only when needed

## Verification

Required verification for this pass:

- `make web-lint`
- `make web-typecheck`

The route itself is the first living proof that the system works as a reusable foundation rather than a one-off page skin.
