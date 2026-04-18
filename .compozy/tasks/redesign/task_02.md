---
status: completed
title: Migrate shadcn batch 1 (Dialog, Popover, Sheet, Tooltip) to @agh/ui
type: refactor
complexity: medium
dependencies:
  - task_01
---

# Task 02: Migrate shadcn batch 1 (Dialog, Popover, Sheet, Tooltip) to @agh/ui

<critical name="redesign-execution">
**Agent:** Run implementation with the designer agent defined in `.claude/agents/designer.md` — **execution mode only** (e.g. `mode: execute`, `execution mode`, "ship it"). **Do not** use plan mode (no `questions_v2`, no brainstorm-only passes) for these tasks.

**Mandatory skills:** `agh-design`, `design-taste-frontend`, `minimalist-ui` — activate before writing or changing UI.

**Design system:** `DESIGN.md` (repo root) is the authoritative design-system spec; tokens and rules there override informal styling.
</critical>

## Overview

Move the dialog-family shadcn primitives out of `web/src/components/ui/` and into `packages/ui/src/components/` as first-class exports of `@agh/ui`, rewriting every importer in `web/src/**` to use the new path. These four primitives share unmount animations that depend on `motion`, so they must land together with `UIProvider` already in place.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST relocate `dialog.tsx`, `popover.tsx`, `sheet.tsx`, `tooltip.tsx` from `web/src/components/ui/` to `packages/ui/src/components/` and export them from `packages/ui/src/index.ts`.
- MUST write matching `.stories.tsx` for each primitive in `packages/ui/src/components/stories/` if not already present; cover open/close, focus trap, and reduced-motion states.
- MUST rewrite every importer in `web/src/**` that references `@/components/ui/{dialog,popover,sheet,tooltip}` to import from `@agh/ui` in the same commit.
- MUST delete the original files in `web/src/components/ui/` after all importers are rewritten; no re-export shims remain.
- MUST integrate `motion`'s `AnimatePresence` where the primitive needs exit animations (Dialog overlay, Sheet slide, Popover fade).
- MUST keep the public prop signatures compatible with existing call sites so no domain component needs prop changes in this task.
- MUST honor `prefers-reduced-motion` via `UIProvider` (task_01) — verified per story.
</requirements>

## Subtasks

- [x] 2.1 Move the four primitive source files to `packages/ui/src/components/`.
- [x] 2.2 Update `packages/ui/src/index.ts` with the four new exports.
- [x] 2.3 Add or update stories for each primitive (variants, open states, focus behavior, reduced-motion).
- [x] 2.4 Run a repo-wide rewrite of `from "@/components/ui/(dialog|popover|sheet|tooltip)"` to `from "@agh/ui"` in `web/src/**`.
- [x] 2.5 Delete the old files from `web/src/components/ui/`.
- [x] 2.6 Verify `make verify` passes and the domain screens still render.

## Implementation Details

TechSpec "Impact Analysis" lists these four under `packages/ui/src/components/` additions and `web/src/components/ui/**` deletions. ADR-001 establishes `@agh/ui` as the sole home for generic primitives. ADR-002 forbids shims.

Dialog in particular needs AnimatePresence for the overlay; the existing shadcn primitive uses `data-state` attributes — preserve that so external call sites do not break, but replace the CSS-only opacity animation with `motion`-driven fade + scale.

### Relevant Files

- `web/src/components/ui/dialog.tsx`, `popover.tsx`, `sheet.tsx`, `tooltip.tsx` — sources to move.
- `packages/ui/src/components/` — destination directory.
- `packages/ui/src/index.ts` — add exports.
- `packages/ui/src/components/stories/` — destination for stories.
- ~15 importers across `web/src/systems/**`, including `tasks-create-modal.tsx`, `settings-save-bar.tsx` and various dialog usages — grep `@/components/ui/dialog` etc. to get the full list.
- **Design references** (read-only, do not edit):
  - `DESIGN.md §4` — dialog + overlay visual spec.
  - `docs/design/design-system/preview/components-buttons.html` — ghost/primary button style used by dialog triggers.
  - `docs/design/design-system/preview/components-nav.html` — popover/tooltip positioning reference.

### Dependent Files

- Any `web/src/**` file currently importing `@/components/ui/{dialog|popover|sheet|tooltip}` will be rewritten in this task.
- Task 08 (close `web/src/components/ui/` folder) depends on this task removing its four files.

### Related ADRs

- [ADR-001: Consolidate UI primitives into @agh/ui](adrs/adr-001.md)
- [ADR-002: Greenfield migration](adrs/adr-002.md)
- [ADR-003: Adopt motion for UI animations](adrs/adr-003.md)

## Deliverables

- Four primitives relocated, exported, stories updated.
- Every importer rewritten; no `@/components/ui/{dialog|popover|sheet|tooltip}` path remains in `web/src/**`.
- Four source files deleted from `web/src/components/ui/`.
- Unit tests with 80%+ coverage for each primitive **(REQUIRED)**.
- Storybook interaction tests for open/close + focus behavior **(REQUIRED)**.

## Tests

- Unit tests:
  - [x] `Dialog` opens on trigger click and closes on `Escape`.
  - [x] `Popover` positions relative to trigger and closes on outside click.
  - [x] `Sheet` enters from the declared side and traps focus while open.
  - [x] `Tooltip` delays show by the configured `delayDuration`.
  - [x] Under `prefers-reduced-motion: reduce` no transform animations run (opacity only).
- Integration tests:
  - [x] Storybook `play()` opens a Dialog, tabs through internal focusables, and closes via overlay click.
  - [x] One existing `web/src/systems/**` screen that used `Dialog` renders and its test suite continues to pass after the import rewrite.
- Test coverage target: >=80%
- All tests must pass

## Success Criteria

- All tests passing
- Test coverage >=80%
- Grep `rg "@/components/ui/(dialog|popover|sheet|tooltip)" web/src` returns zero matches.
- `packages/ui/src/components/{dialog,popover,sheet,tooltip}.tsx` exist and are exported from `src/index.ts`.
- `make verify` passes.
