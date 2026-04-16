---
status: completed
title: "Create packages/ui (design tokens + base components)"
type: refactor
complexity: medium
dependencies: []
---

# Task 01: Create packages/ui (design tokens + base components)

## Overview

Extract the shared design system from `web/src/styles.css` into a new `packages/ui` workspace package. This package becomes the single source of truth for DESIGN.md tokens (CSS custom properties, Tailwind preset, font imports) and server-safe base UI components (Button, Badge, Card, etc.). Both `web/` (Vite SPA) and `packages/site/` (Fumadocs Next.js) will consume `@agh/ui`, ensuring visual consistency without token duplication. See TechSpec "System Architecture > Component Overview" for the package structure and "packages/ui export boundaries" for the server/client split.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
- **IMPECCABLE (non-blocking, Compozy-safe)** — Read and apply the **impeccable** skill (`/impeccable` — already available in Claude Code; do not run any install or `npx` commands). **Read** `.impeccable.md` at repo root when it exists. **Never** run `/impeccable teach` during automated task execution — it is interactive and will stall `compozy start` loops. If `.impeccable.md` is missing, derive constraints only from DESIGN.md, TechSpec, and this task (no user prompts). Follow OKLCH/color and spatial principles, motion rules, and **absolute_bans** (e.g. no thick side-stripe borders on cards/lists/callouts; no gradient fill on text). When lifting patterns into `@agh/ui`, follow the skill’s extract guidance **without** interviews. Where DESIGN.md/TechSpec names concrete AGH tokens or stacks, those specs win.
</critical>

<requirements>
- MUST create `packages/ui/` directory with `package.json` naming the package `@agh/ui`
- MUST extract all DESIGN.md CSS custom properties from `web/src/styles.css` into `packages/ui/src/tokens.css` — this includes `:root` variables (backgrounds, text, accent, semantic, badge tints, shadcn theme mappings), the `@theme inline` block, the `@custom-variant dark` directive, the `@layer base` styles, font imports (`@fontsource-variable/inter`, `@fontsource/jetbrains-mono`), and custom utilities (`animate-shimmer`)
- MUST create `packages/ui/src/tailwind-preset.ts` that exports a Tailwind CSS v4 preset encoding DESIGN.md spacing scale, border radius scale, font families, and color references
- MUST move server-safe base components from `web/src/components/ui/` into `packages/ui/src/components/` — initial set: `button.tsx`, `badge.tsx`, `card.tsx`, `input.tsx`, `label.tsx`, `separator.tsx`, `skeleton.tsx`, `spinner.tsx`, `alert.tsx`, `progress.tsx`, `table.tsx`, `kbd.tsx`
- MUST add `"use client"` directive to any component that uses React hooks (`useState`, `useRef`, `useEffect`) or browser APIs — these are client components and must be explicitly marked for Next.js App Router compatibility
- MUST keep purely presentational components (no hooks, no browser APIs) without `"use client"` so they work as Server Components in Next.js
- MUST create `packages/ui/src/index.ts` as the public API barrel with explicit named exports
- MUST create a `packages/ui/src/lib/utils.ts` with the shared `cn()` utility (clsx + tailwind-merge)
- MUST configure `package.json` with proper `exports` map separating server-safe entries (tokens, preset) from client component entries
- MUST add `packages/ui` to root `package.json` workspaces array
- MUST add `packages/ui` build pipeline to `turbo.json` (output: `dist/**`)
- MUST configure TypeScript (`tsconfig.json`) for the package
- MUST install required dependencies: `class-variance-authority`, `clsx`, `tailwind-merge`, `@fontsource-variable/inter`, `@fontsource/jetbrains-mono`, `tw-animate-css`, `tailwindcss`
- MUST NOT move domain-specific or complex interactive components (sidebar, command, combobox, dialog, sheet, popover, etc.) — those stay in `web/` for alpha
- MUST NOT break `web/` — it continues working with its existing imports until task_02 migrates them
</requirements>

## Subtasks

- [x] 1.1 Create `packages/ui/` directory structure: `src/`, `src/components/`, `src/lib/`
- [x] 1.2 Create `packages/ui/package.json` with `@agh/ui` name, exports map, peer dependencies
- [x] 1.3 Create `packages/ui/tsconfig.json` for TypeScript compilation
- [x] 1.4 Extract CSS tokens from `web/src/styles.css` into `packages/ui/src/tokens.css`
- [x] 1.5 Create `packages/ui/src/tailwind-preset.ts` encoding DESIGN.md scale and tokens
- [x] 1.6 Create `packages/ui/src/lib/utils.ts` with `cn()` utility
- [x] 1.7 Copy and adapt server-safe base components into `packages/ui/src/components/`
- [x] 1.8 Add `"use client"` directives to components that use React hooks
- [x] 1.9 Create `packages/ui/src/index.ts` barrel with explicit named exports
- [x] 1.10 Add `packages/ui` to root `package.json` workspaces and `turbo.json`
- [x] 1.11 Install package dependencies via `bun add`
- [x] 1.12 Verify `turbo run build --filter=@agh/ui` compiles successfully
- [x] 1.13 Verify `web/` still builds independently (no regressions)

## Implementation Details

See TechSpec sections: "System Architecture > Component Overview", "packages/ui export boundaries", "Integration Points > packages/ui", "Impact Analysis".

The token extraction splits `web/src/styles.css` into two concerns: the design system tokens (which move to `packages/ui/src/tokens.css`) and the web-app-specific styling (which stays in `web/`). The key distinction is that tokens.css must be server-safe — no browser API references, pure CSS custom properties and Tailwind theme configuration.

Components that only use CVA (class-variance-authority) and `cn()` for styling are server-safe. Components that use `useState`, `useRef`, `useEffect`, or browser globals need `"use client"`.

### Relevant Files

- `web/src/styles.css` — Source of all CSS tokens to extract (`:root` block lines 10-82, `@theme inline` block lines 88-130, `@layer base` lines 132-147, `@keyframes` and `@utility` lines 149-161)
- `web/src/components/ui/button.tsx` — Base component to move
- `web/src/components/ui/badge.tsx` — Base component to move
- `web/src/components/ui/card.tsx` — Base component to move
- `web/src/components/ui/input.tsx` — Base component to move
- `web/src/components/ui/label.tsx` — Base component to move
- `web/src/components/ui/separator.tsx` — Base component to move
- `web/src/components/ui/skeleton.tsx` — Base component to move
- `web/src/components/ui/spinner.tsx` — Base component to move
- `web/src/components/ui/alert.tsx` — Base component to move
- `web/src/components/ui/progress.tsx` — Base component to move
- `web/src/components/ui/table.tsx` — Base component to move
- `web/src/components/ui/kbd.tsx` — Base component to move
- `web/src/lib/utils.ts` — Contains `cn()` utility to duplicate into packages/ui
- `DESIGN.md` — Canonical design system reference for all token values
- `turbo.json` — Add `packages/ui` workspace config
- `package.json` (root) — Add `packages/ui` to workspaces array

### Dependent Files

- `web/src/styles.css` — Will be updated to import from `@agh/ui` (task_02)
- `web/src/components/ui/*` — Will be replaced with re-exports from `@agh/ui` (task_02)
- `packages/site/` — Will consume `@agh/ui` tokens and components (task_03)

### Related ADRs

- [ADR-002: Monorepo Package Layout](adrs/adr-002.md) — Defines packages/ui as shared design token + component package alongside existing web/

## Deliverables

- `packages/ui/package.json` with `@agh/ui` name, exports map, dependencies
- `packages/ui/tsconfig.json`
- `packages/ui/src/tokens.css` with all DESIGN.md CSS custom properties
- `packages/ui/src/tailwind-preset.ts` with Tailwind CSS v4 preset
- `packages/ui/src/lib/utils.ts` with `cn()` utility
- `packages/ui/src/components/` with 12 server-safe base components
- `packages/ui/src/index.ts` barrel export
- Updated root `package.json` workspaces
- Updated `turbo.json` with packages/ui pipeline
- All builds passing: `turbo run build --filter=@agh/ui` and `make web-build`

## Tests

- Build verification:
  - [ ] `turbo run build --filter=@agh/ui` completes without errors
  - [ ] `make web-build` still passes (no regressions to web/)
  - [ ] `make web-typecheck` still passes
- Package structure:
  - [ ] `packages/ui/src/tokens.css` contains all `:root` custom properties from DESIGN.md
  - [ ] `packages/ui/src/tokens.css` includes font imports, `@theme inline`, `@layer base`, and `@keyframes`
  - [ ] `packages/ui/src/tailwind-preset.ts` exports a valid Tailwind preset
  - [ ] All 12 base components exist in `packages/ui/src/components/`
  - [ ] Components using React hooks have `"use client"` directive
  - [ ] `packages/ui/src/index.ts` exports all components and utilities
- Export boundary:
  - [ ] `tokens.css` and `tailwind-preset.ts` contain no browser API references
  - [ ] `cn()` utility works in both server and client contexts
- Test coverage target: N/A (package scaffolding — verified by build, not unit tests)

## Success Criteria

- `turbo run build --filter=@agh/ui` succeeds
- `make web-build` succeeds (zero regressions)
- Root `package.json` includes `packages/ui` in workspaces
- `turbo.json` has `packages/ui` in workspace pipeline
- All DESIGN.md tokens present in `packages/ui/src/tokens.css`
- 12 base components extracted with correct `"use client"` directives
- Package `exports` map correctly separates server-safe and client entries
