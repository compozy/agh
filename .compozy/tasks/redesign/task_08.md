---
status: pending
title: Close web/src/components/ui/ — migrate remaining shadcn primitives and delete folder
type: refactor
complexity: high
dependencies:
  - task_01
  - task_02
  - task_03
  - task_04
  - task_05
  - task_06
---

# Task 08: Close web/src/components/ui/ — migrate remaining shadcn primitives and delete folder

<critical name="redesign-execution">
**Agent:** Run implementation with the designer agent defined in `.claude/agents/designer.md` — **execution mode only** (e.g. `mode: execute`, `execution mode`, "ship it"). **Do not** use plan mode (no `questions_v2`, no brainstorm-only passes) for these tasks.

**Mandatory skills:** `agh-design`, `design-taste-frontend`, `minimalist-ui` — activate before writing or changing UI.

**Design system:** `DESIGN.md` (repo root) is the authoritative design-system spec; tokens and rules there override informal styling.
</critical>

## Overview

Migrate the last remaining primitives in `web/src/components/ui/` (Avatar, Breadcrumb, ButtonGroup, Field, InputGroup, Item, NativeSelect, Textarea, Sonner, Direction) to `@agh/ui` and remove the folder entirely. With this task the consolidation promise in ADR-001 is complete: every generic primitive lives in `@agh/ui`.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST relocate `avatar.tsx`, `breadcrumb.tsx`, `button-group.tsx`, `field.tsx`, `input-group.tsx`, `item.tsx`, `native-select.tsx`, `textarea.tsx`, `sonner.tsx`, `direction.tsx` from `web/src/components/ui/` to `packages/ui/src/components/` and export from `packages/ui/src/index.ts`.
- MUST rewrite every `@/components/ui/(avatar|breadcrumb|button-group|field|input-group|item|native-select|textarea|sonner|direction)` importer in `web/src/**` to `@agh/ui`.
- MUST add stories covering each primitive's variants (Field error state, InputGroup addons, Avatar fallback, ButtonGroup separators, Item with media + actions, Sonner toast variants).
- MUST DELETE the entire `web/src/components/ui/` directory after all files are moved; any residual file fails the task.
- MUST ensure `tsconfig.json` path aliases no longer reference `@/components/ui/*` (or rewire them if they still serve).
- SHOULD ensure Sonner's toast provider is either re-exported from `@agh/ui` or mounted at the app root with a single call — no multiple mounts.
</requirements>

## Subtasks

- [ ] 8.1 Move the ten primitive source files to `packages/ui/src/components/`.
- [ ] 8.2 Update `packages/ui/src/index.ts` with all ten new exports and any sub-exports (FieldError, InputGroupAddon, ItemMedia, etc.).
- [ ] 8.3 Add or update stories for each.
- [ ] 8.4 Rewrite every remaining `@/components/ui/*` import in `web/src/**` to `@agh/ui` (after tasks 02–04, 06 only these ten paths should remain).
- [ ] 8.5 Delete the entire `web/src/components/ui/` directory and clean up any tsconfig path aliases that pointed to it.
- [ ] 8.6 Verify `rg "@/components/ui/" web/src` returns zero matches and `make verify` passes.

## Implementation Details

See ADR-001 for the closure rationale and TechSpec "Impact Analysis" for the folder deletion row. Most of these primitives are one-importer or five-importer (per inventory); Textarea has ~5, Field has ~5, InputGroup has ~5, NativeSelect has ~5. All are low-risk individually but the folder deletion is the tracking milestone.

### Relevant Files

- `web/src/components/ui/*` — ten remaining source files.
- `packages/ui/src/components/` — destination.
- `packages/ui/src/index.ts` — export list.
- `web/tsconfig.json` — path aliases (check for `@/components/ui/*`).
- Grep `@/components/ui/(avatar|breadcrumb|button-group|field|input-group|item|native-select|textarea|sonner|direction)`.

### Dependent Files

- `web/src/systems/settings/**` (uses Field, NativeSelect, Switch combinations).
- Task 13 (app-sidebar rewrite) depends on this folder being gone.

### Related ADRs

- [ADR-001: Consolidate UI primitives into @agh/ui](adrs/adr-001.md)
- [ADR-002: Greenfield migration](adrs/adr-002.md)

## Deliverables

- Ten primitives relocated, exported, stories updated.
- All importers rewritten.
- `web/src/components/ui/` directory deleted.
- Unit tests with 80%+ coverage for each primitive **(REQUIRED)**.
- Storybook interaction tests for Field error handling + Sonner toast display **(REQUIRED)**.

## Tests

- Unit tests:
  - [ ] `Field` renders its label, description, error state with correct `aria-describedby` wiring.
  - [ ] `InputGroup` places prefix + suffix addons without clipping the input.
  - [ ] `Avatar` falls back to initials when image load fails.
  - [ ] `ButtonGroup` renders separators between direct Button children.
  - [ ] `Item` composes media + content + actions + separator in the declared slot order.
  - [ ] `NativeSelect` forwards value + onChange correctly.
  - [ ] `Textarea` supports `rows` + autoresize if enabled.
  - [ ] `Sonner` toast API (`toast.success`, `toast.error`) mounts at the configured position.
  - [ ] `Direction` provider forwards `dir` to Radix components.
- Integration tests:
  - [ ] Storybook `play()` triggers a `Sonner.toast.error` and asserts the toast renders with danger tone.
  - [ ] A settings form using Field + Switch continues to save unchanged after the import rewrite.
- Test coverage target: >=80%
- All tests must pass

## Success Criteria

- All tests passing
- Test coverage >=80%
- `rg "@/components/ui/" web/src` returns zero matches.
- `web/src/components/ui/` directory does not exist.
- All ten primitives exported from `packages/ui/src/index.ts`.
- `make verify` passes.
