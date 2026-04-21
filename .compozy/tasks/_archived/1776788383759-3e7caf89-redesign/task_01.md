---
status: completed
title: Extend tokens, install motion, add UIProvider
type: frontend
complexity: low
dependencies: []
---

# Task 01: Extend tokens, install motion, add UIProvider

<critical name="redesign-execution">
**Agent:** Run implementation with the designer agent defined in `.claude/agents/designer.md` — **execution mode only** (e.g. `mode: execute`, `execution mode`, "ship it"). **Do not** use plan mode (no `questions_v2`, no brainstorm-only passes) for these tasks.

**Mandatory skills:** `agh-design`, `design-taste-frontend`, `minimalist-ui` — activate before writing or changing UI.

**Design system:** `DESIGN.md` (repo root) is the authoritative design-system spec; tokens and rules there override informal styling.
</critical>

## Overview

Establish the foundation of the new `@agh/ui` surface: add missing tokens from `DESIGN.md` to `packages/ui/src/tokens.css`, install the `motion` package as a peer/dev dep, and expose a single `UIProvider` that wires `MotionConfig` with `reducedMotion="user"`. Every subsequent task in Phase 1 builds on these additions.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST add the following tokens to `packages/ui/src/tokens.css`: `--radius-chip` (5px), `--radius-mono-badge` (6px), `--font-display` (Playfair Display stack), `--font-wordmark` (NuixyberNext stack), `--duration-fast` (100ms), `--duration-base` (150ms), `--duration-slow` (200ms), `--ease-out` (`cubic-bezier(0.2, 0, 0, 1)`), `--ease-in-out` (`cubic-bezier(0.4, 0, 0.2, 1)`), `--color-accent-tint-strong` (`#E8572A3D`).
- MUST NOT change the value of any existing token — only additions are allowed.
- MUST add `motion` as peer + dev dep in `packages/ui/package.json` and as runtime dep in `web/package.json` via `pnpm add`.
- MUST export a `UIProvider` component from `packages/ui/src/index.ts` that wraps `MotionConfig reducedMotion="user"` and a default transition of 150ms ease-out.
- MUST add a Storybook story for `UIProvider` demonstrating the reducedMotion override contract.
- MUST trim `web/src/styles.css` so it only imports `@agh/ui/tokens.css` + any essential globals; drop any local primitive class definitions and dead CSS that will be replaced by `@agh/ui` primitives later.
- SHOULD keep the file additions under 60 LOC (provider) and under 40 new lines in tokens.css.
</requirements>

## Subtasks

- [x] 1.1 Add the 10 new tokens to `packages/ui/src/tokens.css` in the appropriate section blocks (radii, typography, motion, color).
- [x] 1.2 Run `pnpm add motion` in both `packages/ui` (peer + dev) and `web/`; pin to a known minor version.
- [x] 1.3 Create `packages/ui/src/components/ui-provider.tsx` with the MotionConfig wrapper and export it from `src/index.ts`.
- [x] 1.4 Write `packages/ui/src/components/stories/ui-provider.stories.tsx` with a variant demonstrating `reducedMotion="always"`.
- [x] 1.5 Trim `web/src/styles.css` to imports + globals; remove dead primitive class definitions.
- [x] 1.6 Verify `pnpm --filter @agh/ui build` succeeds and `pnpm --filter @agh/ui test` passes with the new story rendering.

## Implementation Details

See TechSpec "Core Interfaces" for `UIProviderProps` shape and "Data Models" → token additions table.

Motion library choice is in ADR-003.

### Relevant Files

- `packages/ui/src/tokens.css` — token definitions; additions append to existing sections.
- `packages/ui/package.json` — dependency + peer dep updates.
- `packages/ui/src/index.ts` — public exports; add `UIProvider`.
- `web/package.json` — runtime dep for `motion`.
- `web/src/styles.css` — trim to imports + globals.
- **Design references** (read-only, do not edit):
  - `DESIGN.md` — repo-root authoritative spec for tokens, type scale, motion.
  - `docs/design/design-system/colors_and_type.css` — mirror of `@agh/ui/tokens.css` plus the `--font-display`, `--font-wordmark`, `--tracking-eyebrow` tokens and the clamp-based display typography classes. Confirm the ten tokens being added match this file.
  - `docs/design/design-system/preview/colors-surfaces.html`, `preview/colors-accent.html`, `preview/colors-semantic.html`, `preview/colors-text.html` — token swatch references.
  - `docs/design/design-system/preview/spacing-radii.html`, `preview/spacing-elevation.html` — radius + elevation token references.
  - `docs/design/design-system/preview/type-display.html`, `preview/type-body.html`, `preview/type-mono.html`, `preview/type-docs-heading.html` — typography token references.

### Dependent Files

- Every subsequent Phase 1 task consumes the new tokens and MotionConfig.
- `web/src/main.tsx` (touched later in task 14) will import and wrap the app in `<UIProvider>`.

### Related ADRs

- [ADR-003: Adopt motion for UI animations](adrs/adr-003.md) — justifies lib choice and `reducedMotion="user"` default.

## Deliverables

- 10 new CSS variables in `packages/ui/src/tokens.css`.
- `motion` installed in both packages, pinned minor version.
- `UIProvider` component + export + story.
- Unit tests with 80%+ coverage for `UIProvider` **(REQUIRED)**.
- Storybook interaction test covering the `reducedMotion` override **(REQUIRED)**.

## Tests

- Unit tests:
  - [ ] `UIProvider` renders children without crashing under the default config.
  - [ ] `UIProvider` with `reducedMotion="always"` forwards the setting to `MotionConfig` (assert on the consumer via `useReducedMotion`).
  - [ ] `UIProvider` defaults to `reducedMotion="user"` when the prop is omitted.
- Integration tests:
  - [ ] Storybook play function toggles `reducedMotion` and verifies a motion child honors the setting (no transform during reduced motion).
- Test coverage target: >=80%
- All tests must pass

## Success Criteria

- All tests passing
- Test coverage >=80%
- `packages/ui/src/tokens.css` contains all 10 new tokens with values exactly matching DESIGN.md.
- `pnpm --filter @agh/ui build` produces a bundle that exports `UIProvider`.
- `pnpm --filter web build` continues to succeed (`UIProvider` not yet wired into the app — that happens in task 14).
