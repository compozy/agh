---
status: completed
title: "Update web/ to consume packages/ui"
type: refactor
complexity: medium
dependencies: [task_01]
---

# Task 02: Update web/ to consume packages/ui

## Overview

Migrate the existing `web/` React SPA to consume design tokens and base UI components from `@agh/ui` instead of its own local copies. After this task, `web/src/styles.css` imports tokens from the shared package, and all base component imports point to `@agh/ui`. The local component files in `web/src/components/ui/` for moved components are either deleted or converted to thin re-exports. See TechSpec "Impact Analysis" for the scope of changes to `web/`.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
- **IMPECCABLE (non-blocking, Compozy-safe)** — Preserve visual parity; any styling fix must follow the **impeccable** skill (`/impeccable` — already in Claude Code; no installs). Use `.impeccable.md` when present; otherwise DESIGN.md + TechSpec + this task. **Never** run `/impeccable teach` in unattended execution. Apply **absolute_bans** and anti-slop checks; do not introduce thick side-stripe borders, gradient text, or forbidden patterns.
</critical>

<requirements>
- MUST add `@agh/ui` as a workspace dependency in `web/package.json`
- MUST update `web/src/styles.css` to import tokens from `@agh/ui/tokens.css` instead of declaring them inline — remove the extracted `:root` block, `@theme inline` block, `@layer base`, font imports, and `@keyframes`/`@utility` that now live in `packages/ui/src/tokens.css`
- MUST keep any web-specific styles that are NOT part of the shared design system (e.g., `#app` min-height, Vite-specific config)
- MUST update all imports of the 12 base components (button, badge, card, input, label, separator, skeleton, spinner, alert, progress, table, kbd) across `web/src/` to import from `@agh/ui` instead of `@/components/ui/`
- MUST either delete the local copies of moved components from `web/src/components/ui/` or convert them to single-line re-exports from `@agh/ui` — prefer deletion since this is greenfield alpha with zero legacy tolerance
- MUST NOT touch components that remain in `web/src/components/ui/` (sidebar, command, combobox, dialog, sheet, popover, tooltip, accordion, breadcrumb, collapsible, dropdown-menu, field, select, scroll-area, sonner, switch, tabs, textarea, toggle, toggle-group, button-group, direction, empty, input-group, item, native-select, avatar)
- MUST update `web/src/lib/utils.ts` to re-export `cn` from `@agh/ui` or keep as a local copy if it has web-specific additions
- MUST verify `make web-build`, `make web-lint`, `make web-typecheck`, and `make web-test` all pass after migration
- MUST verify that the web dev server starts and renders correctly
</requirements>

## Subtasks

- [x] 2.1 Add `@agh/ui` workspace dependency to `web/package.json`
- [x] 2.2 Update `web/src/styles.css` to import tokens from `@agh/ui`
- [x] 2.3 Find all imports of the 12 moved components across `web/src/`
- [x] 2.4 Update each import to reference `@agh/ui` instead of local path
- [x] 2.5 Delete local copies of moved components from `web/src/components/ui/`
- [x] 2.6 Update `web/src/lib/utils.ts` to use `cn` from `@agh/ui`
- [x] 2.7 Run `bun install` to link workspace dependencies
- [x] 2.8 Verify `make web-build` passes
- [x] 2.9 Verify `make web-lint` passes
- [x] 2.10 Verify `make web-typecheck` passes
- [x] 2.11 Verify `make web-test` passes

## Implementation Details

See TechSpec sections: "Impact Analysis — web/", "Integration Points > packages/ui".

The migration is mechanical: find-and-replace import paths. The critical detail is that `web/src/styles.css` currently contains both design-system tokens (which moved to `@agh/ui`) and web-app boilerplate (which stays). After the split, `web/src/styles.css` should import `@agh/ui/tokens.css` at the top and only retain web-specific additions.

Components remaining in `web/src/components/ui/` that depend on moved components (e.g., `combobox.tsx` importing `button.tsx`) must have their imports updated to point to `@agh/ui` as well.

### Relevant Files

- `web/package.json` — Add `@agh/ui` workspace dependency
- `web/src/styles.css` — Replace inline tokens with import from `@agh/ui/tokens.css`
- `web/src/components/ui/button.tsx` — Delete (moved to packages/ui)
- `web/src/components/ui/badge.tsx` — Delete (moved to packages/ui)
- `web/src/components/ui/card.tsx` — Delete (moved to packages/ui)
- `web/src/components/ui/input.tsx` — Delete (moved to packages/ui)
- `web/src/components/ui/label.tsx` — Delete (moved to packages/ui)
- `web/src/components/ui/separator.tsx` — Delete (moved to packages/ui)
- `web/src/components/ui/skeleton.tsx` — Delete (moved to packages/ui)
- `web/src/components/ui/spinner.tsx` — Delete (moved to packages/ui)
- `web/src/components/ui/alert.tsx` — Delete (moved to packages/ui)
- `web/src/components/ui/progress.tsx` — Delete (moved to packages/ui)
- `web/src/components/ui/table.tsx` — Delete (moved to packages/ui)
- `web/src/components/ui/kbd.tsx` — Delete (moved to packages/ui)
- `web/src/lib/utils.ts` — Update `cn` import source
- All files in `web/src/` that import from `@/components/ui/{button,badge,card,...}` — update import paths

### Dependent Files

- `packages/ui/` — Must be built before web/ can consume it (task_01 deliverable)

### Related ADRs

- [ADR-002: Monorepo Package Layout](adrs/adr-002.md) — web/ consuming packages/ui is the core of the additive package strategy

## Deliverables

- Updated `web/package.json` with `@agh/ui` dependency
- Updated `web/src/styles.css` importing tokens from `@agh/ui`
- All 12 local component files deleted from `web/src/components/ui/`
- All component imports across `web/src/` updated to `@agh/ui`
- Updated `web/src/lib/utils.ts`
- All build, lint, typecheck, and test gates passing

## Tests

- Build verification:
  - [ ] `make web-build` passes with zero errors
  - [ ] `make web-lint` passes with zero warnings
  - [ ] `make web-typecheck` passes
  - [ ] `make web-test` passes
- Import verification:
  - [ ] No remaining imports of `@/components/ui/button` (or other moved components) in `web/src/`
  - [ ] `web/src/components/ui/` no longer contains files for the 12 moved components
  - [ ] Remaining components in `web/src/components/ui/` that import moved components use `@agh/ui` paths
- Visual verification:
  - [ ] `make web-dev` starts without errors
  - [ ] Design tokens (colors, fonts, spacing) render identically to pre-migration
- Test coverage target: N/A (migration — verified by existing test suite passing)

## Success Criteria

- `make web-build` passes
- `make web-lint` passes with zero warnings
- `make web-typecheck` passes
- `make web-test` passes
- Zero imports of `@/components/ui/{button,badge,card,input,label,separator,skeleton,spinner,alert,progress,table,kbd}` remain in `web/src/`
- `web/src/styles.css` imports tokens from `@agh/ui` instead of declaring them inline
- 12 local component files removed from `web/src/components/ui/`
- Dev server starts and renders correctly
