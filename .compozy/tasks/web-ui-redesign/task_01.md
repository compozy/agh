---
status: pending
title: Replace design tokens and fonts
type: frontend
complexity: high
dependencies: []
---

# Task 01: Replace design tokens and fonts

## Overview

Replace the entire CSS design token system in `styles.css` — swapping OKLCH colors for DESIGN.md hex values, Geist/Bricolage fonts for Inter/JetBrains Mono, and removing all gradients, textures, and shadows to achieve the flat depth model. This is the foundation that every subsequent task depends on.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC "Design Token Replacement" section for exact token values and shadcn mapping
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST replace all OKLCH color values with hex values from DESIGN.md
- MUST swap font imports: remove `@fontsource-variable/geist` and `@fontsource/bricolage-grotesque`, add `@fontsource-variable/inter` and `@fontsource/jetbrains-mono`
- MUST remove `--font-display` theme variable — no display/heading font per DESIGN.md
- MUST remove all `.ds-texture-*`, `.ds-panel*` CSS classes and their `::before`/`::after` pseudo-elements
- MUST remove all `box-shadow`, `color-mix()`, gradient, and vignette declarations
- MUST update shadcn theme variable mappings per TechSpec table (e.g., `--primary: #E8572A`)
- MUST update all existing component files that reference `--ds-*` variables to use new token names
- MUST pass `make web-lint && make web-typecheck` after changes
</requirements>

## Subtasks
- [ ] 1.1 Swap font packages in `package.json` via `bun add`/`bun remove`
- [ ] 1.2 Rewrite `styles.css` with DESIGN.md hex tokens, new font theme, flat depth model
- [ ] 1.3 Update all component files that reference `--ds-*` CSS variables to use new token names
- [ ] 1.4 Remove references to `font-display` and `ds-texture-canvas` classes from components
- [ ] 1.5 Verify `make web-lint && make web-typecheck` passes with zero errors
- [ ] 1.6 Write tests verifying no OKLCH values, no shadow declarations remain

## Implementation Details

See TechSpec "Design Token Replacement" section for the complete token mapping table, shadcn variable mapping, and font theme definition.

Key changes:
- `web/src/styles.css`: Complete rewrite of `:root` block, `@theme inline` block, and removal of `.ds-*` utility classes
- `web/package.json`: Font dependency swap
- All files referencing `var(--ds-*)` patterns need mechanical find-replace to new token names

### Relevant Files
- `web/src/styles.css` — Main design token file, complete rewrite target
- `web/package.json` — Font package dependencies to swap
- `web/src/components/app-sidebar.tsx` — References `--ds-text-muted`, `--ds-text-mono`, `--ds-panel-accent`, `--ds-accent-amber`
- `web/src/components/app-header.tsx` — References `--ds-*` variables
- `web/src/routes/_app.tsx` — Uses `ds-texture-canvas-subtle` class
- `web/src/routes/_app/index.tsx` — References `--ds-text-muted`
- `web/src/systems/session/components/` — Multiple components reference `--ds-*` vars
- `web/src/systems/daemon/components/connection-status.tsx` — References `--ds-*` vars

### Dependent Files
- Every component in `web/src/` that uses CSS variables — styling will change
- `web/src/components/ui/sidebar.tsx` — shadcn sidebar theme vars will update automatically via `:root` changes
- `web/src/components/design-system/` — Showcase components may need removal or update

### Related ADRs
- [ADR-001: Full Replace of Design Token System](../adrs/adr-001.md) — Mandates full replacement, no migration layer

## Deliverables
- Rewritten `styles.css` with all DESIGN.md tokens
- Updated `package.json` with Inter + JetBrains Mono fonts
- All existing components updated to reference new tokens
- Zero OKLCH values, zero shadows, zero gradients in CSS
- Unit tests with 80%+ coverage **(REQUIRED)**

## Tests
- Unit tests:
  - [ ] `styles.css` contains no `oklch(` values
  - [ ] `styles.css` contains no `box-shadow` declarations
  - [ ] `styles.css` contains no `color-mix(` expressions
  - [ ] `styles.css` contains no `.ds-texture-` class definitions
  - [ ] `styles.css` defines `--color-canvas: #121212`
  - [ ] `styles.css` defines `--color-accent: #E8572A`
  - [ ] `@theme inline` declares `--font-sans` with "Inter Variable"
  - [ ] `@theme inline` declares `--font-mono` with "JetBrains Mono"
  - [ ] `@theme inline` does NOT declare `--font-display`
  - [ ] No component file contains `var(--ds-` references
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- `make web-lint` passes with zero warnings
- `make web-typecheck` passes with zero errors
- App renders in browser with new visual language (hex colors, Inter/JetBrains Mono, flat depth)
