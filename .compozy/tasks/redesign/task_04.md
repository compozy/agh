---
status: completed
title: Migrate shadcn batch 3 (DropdownMenu, Switch, Toggle, ToggleGroup, Accordion, Collapsible) to @agh/ui
type: refactor
complexity: medium
dependencies:
  - task_01
---

# Task 04: Migrate shadcn batch 3 (DropdownMenu, Switch, Toggle, ToggleGroup, Accordion, Collapsible) to @agh/ui

<critical name="redesign-execution">
**Agent:** Run implementation with the designer agent defined in `.claude/agents/designer.md` — **execution mode only** (e.g. `mode: execute`, `execution mode`, "ship it"). **Do not** use plan mode (no `questions_v2`, no brainstorm-only passes) for these tasks.

**Mandatory skills:** `agh-design`, `design-taste-frontend`, `minimalist-ui` — activate before writing or changing UI.

**Design system:** `DESIGN.md` (repo root) is the authoritative design-system spec; tokens and rules there override informal styling.
</critical>

## Overview

Move the remaining state-carrying shadcn primitives into `@agh/ui`. These primitives back settings toggles, expandable sidebar sections, and context menus across `web/`. Switch alone has ~10 importers (see inventory), so the rewrite must be done in one atomic commit.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST relocate `dropdown-menu.tsx`, `switch.tsx`, `toggle.tsx`, `toggle-group.tsx`, `accordion.tsx`, `collapsible.tsx` from `web/src/components/ui/` to `packages/ui/src/components/` and export them from `packages/ui/src/index.ts`.
- MUST write stories covering: Switch (disabled, checked, sizes), Toggle (pressed/unpressed), ToggleGroup (single/multiple mode), Accordion (single/multiple, collapsible), Collapsible (open/closed transition), DropdownMenu (nested submenus, checkbox items, radio items).
- MUST rewrite every importer in `web/src/**` to use `@agh/ui` and delete the originals in the same PR.
- MUST preserve existing prop signatures so settings forms and sidebar expand/collapse continue working unchanged.
- SHOULD replace CSS-only accordion height animation with `motion` height animation only if it simplifies the code — otherwise keep current `data-state` CSS approach.
</requirements>

## Subtasks

- [x] 4.1 Move the six primitive source files to `packages/ui/src/components/`.
- [x] 4.2 Update `packages/ui/src/index.ts` with the six new exports and their sub-exports.
- [x] 4.3 Add or update stories with required variants.
- [x] 4.4 Rewrite every `@/components/ui/(dropdown-menu|switch|toggle|toggle-group|accordion|collapsible)` import in `web/src/**` to `@agh/ui`.
- [x] 4.5 Delete the moved files from `web/src/components/ui/`.
- [x] 4.6 Run `make verify` and fix any breakages.

## Implementation Details

See TechSpec "Impact Analysis" and ADR-001 for framing. Switch with ~10 importers is the highest-risk file in this batch — audit call sites before starting. Most live under `web/src/systems/settings/components/**`.

### Relevant Files

- `web/src/components/ui/dropdown-menu.tsx`, `switch.tsx`, `toggle.tsx`, `toggle-group.tsx`, `accordion.tsx`, `collapsible.tsx` — sources.
- `packages/ui/src/components/` — destination.
- `packages/ui/src/index.ts` — export list.
- Grep `@/components/ui/(dropdown-menu|switch|toggle|toggle-group|accordion|collapsible)` for call sites.
- **Design references** (read-only, do not edit):
  - `DESIGN.md §4` — toggle + menu primitives spec.
  - `docs/design/design-system/preview/components-buttons.html` — toggle visual treatment.
  - `docs/design/design-system/preview/components-inputs.html` — switch affordance.

### Dependent Files

- Task 08 (folder close-out) depends on these six files being gone.
- `web/src/systems/settings/components/**` uses Switch heavily.

### Related ADRs

- [ADR-001: Consolidate UI primitives into @agh/ui](adrs/adr-001.md)
- [ADR-002: Greenfield migration](adrs/adr-002.md)

## Deliverables

- Six primitives relocated, exported, stories updated.
- All importers rewritten; originals deleted.
- Unit tests with 80%+ coverage for each primitive **(REQUIRED)**.
- Storybook interaction tests for toggle + accordion state changes **(REQUIRED)**.

## Tests

- Unit tests:
  - [x] `Switch` toggles between checked and unchecked on click.
  - [x] `Switch` fires `onCheckedChange` with the new value.
  - [x] `Accordion` closes the previous item when a new one opens (Base UI single-selection — default `multiple={false}`).
  - [x] `Collapsible` preserves content in the DOM when closed with `keepMounted` and transitions open/closed states.
  - [x] `DropdownMenu` opens via click and forwards selection events (checkbox/radio/plain items). Submenu wiring is covered by the `WithSubmenu` story.
  - [x] `ToggleGroup` with `multiple` accumulates pressed items.
- Integration tests:
  - [x] Storybook `play()` for Accordion opens an item, asserts aria-expanded, and closes it (`OpensAndCloses` story).
  - [x] `web/src/routes/_app/settings/memory.tsx` (and sibling settings routes) continue to wire Switch → draft state → save through `SettingsSaveBar`, verified by the green `make web-test` suite.
- Test coverage target: >=80%
- All tests must pass

## Success Criteria

- All tests passing
- Test coverage >=80%
- `rg "@/components/ui/(dropdown-menu|switch|toggle|toggle-group|accordion|collapsible)" web/src` returns zero matches.
- All six primitives exported from `packages/ui/src/index.ts`.
- `make verify` passes.
