---
status: pending
title: Migrate shadcn batch 2 (Combobox, Command, Select, ScrollArea, Tabs) to @agh/ui
type: refactor
complexity: medium
dependencies:
  - task_01
---

# Task 03: Migrate shadcn batch 2 (Combobox, Command, Select, ScrollArea, Tabs) to @agh/ui

<critical name="redesign-execution">
**Agent:** Run implementation with the designer agent defined in `.claude/agents/designer.md` — **execution mode only** (e.g. `mode: execute`, `execution mode`, "ship it"). **Do not** use plan mode (no `questions_v2`, no brainstorm-only passes) for these tasks.

**Mandatory skills:** `agh-design`, `design-taste-frontend`, `minimalist-ui` — activate before writing or changing UI.

**Design system:** `DESIGN.md` (repo root) is the authoritative design-system spec; tokens and rules there override informal styling.
</critical>

## Overview

Move the selection-family shadcn primitives out of `web/src/components/ui/` and into `packages/ui/src/components/`, rewriting every importer to the new path. These primitives expose richer APIs (multi-select, searchable, grouped options) and need stories that exercise the full surface.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST relocate `combobox.tsx`, `command.tsx`, `select.tsx`, `scroll-area.tsx`, `tabs.tsx` from `web/src/components/ui/` to `packages/ui/src/components/` and export them from `packages/ui/src/index.ts`.
- MUST write matching stories covering: default, multi-select (Combobox), grouped options (Select), horizontal + vertical orientations (Tabs), overflow (ScrollArea), keyboard navigation (Command).
- MUST rewrite every importer in `web/src/**` to resolve through `@agh/ui` and delete the originals in the same PR.
- MUST preserve existing public prop signatures so no domain code needs refactoring in this task.
- MUST keep keyboard accessibility (Arrow/Enter/Esc/Tab) identical; verified per story.
</requirements>

## Subtasks

- [ ] 3.1 Move the five primitive source files to `packages/ui/src/components/`.
- [ ] 3.2 Update `packages/ui/src/index.ts` with the five new exports plus any associated sub-exports (e.g., `SelectGroup`, `CommandItem`).
- [ ] 3.3 Add or update stories with the required variants above.
- [ ] 3.4 Rewrite every `@/components/ui/(combobox|command|select|scroll-area|tabs)` import in `web/src/**` to `@agh/ui`.
- [ ] 3.5 Delete the moved files from `web/src/components/ui/`.
- [ ] 3.6 Run `make verify` and fix any breakages.

## Implementation Details

See TechSpec "Impact Analysis" and ADR-001 for context. Combobox is currently a custom wrapper around Popover + Command (see `web/src/components/ui/combobox.tsx`); after migration it still composes `@agh/ui`'s Popover + Command which are colocated in the same package.

### Relevant Files

- `web/src/components/ui/combobox.tsx`, `command.tsx`, `select.tsx`, `scroll-area.tsx`, `tabs.tsx` — sources.
- `packages/ui/src/components/` — destination.
- `packages/ui/src/index.ts` — export list.
- `packages/ui/src/components/stories/` — destination for stories.
- Grep `@/components/ui/(combobox|command|select|scroll-area|tabs)` for all call sites.

### Dependent Files

- Task 08 (folder close-out) depends on these five files being gone.
- Task 13 (app-sidebar rewrite) may use Tabs + ScrollArea and benefits from these landing first.

### Related ADRs

- [ADR-001: Consolidate UI primitives into @agh/ui](adrs/adr-001.md)
- [ADR-002: Greenfield migration](adrs/adr-002.md)

## Deliverables

- Five primitives relocated, exported, stories updated.
- All importers rewritten; originals deleted.
- Unit tests with 80%+ coverage for each primitive **(REQUIRED)**.
- Storybook interaction tests for keyboard navigation + multi-select **(REQUIRED)**.

## Tests

- Unit tests:
  - [ ] `Combobox` in multi-select mode accumulates selections and emits the correct array on change.
  - [ ] `Command` filters items as the user types and selects the highlighted item on Enter.
  - [ ] `Select` opens, navigates with Arrow keys, and closes on Escape.
  - [ ] `ScrollArea` renders a custom track/thumb when content overflows.
  - [ ] `Tabs` orients horizontally by default and honors `orientation="vertical"`.
- Integration tests:
  - [ ] Storybook `play()` for `Command` filters a canned dataset and selects an item via keyboard.
  - [ ] One existing domain screen using `Combobox` renders its test suite unchanged after the import rewrite.
- Test coverage target: >=80%
- All tests must pass

## Success Criteria

- All tests passing
- Test coverage >=80%
- `rg "@/components/ui/(combobox|command|select|scroll-area|tabs)" web/src` returns zero matches.
- All five primitives exported from `packages/ui/src/index.ts`.
- `make verify` passes.
