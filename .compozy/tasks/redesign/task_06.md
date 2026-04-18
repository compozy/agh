---
status: completed
title: Add PageHeader, Pills, SearchInput, Empty, Section, Toolbar and migrate design-system primitives
type: refactor
complexity: critical
dependencies:
  - task_01
---

# Task 06: Add PageHeader, Pills, SearchInput, Empty, Section, Toolbar and migrate design-system primitives

<critical name="redesign-execution">
**Agent:** Run implementation with the designer agent defined in `.claude/agents/designer.md` — **execution mode only** (e.g. `mode: execute`, `execution mode`, "ship it"). **Do not** use plan mode (no `questions_v2`, no brainstorm-only passes) for these tasks.

**Mandatory skills:** `agh-design`, `design-taste-frontend`, `minimalist-ui` — activate before writing or changing UI.

**Design system:** `DESIGN.md` (repo root) is the authoritative design-system spec; tokens and rules there override informal styling.
</critical>

## Overview

Introduce the six page-level primitives from the mock and migrate the legacy `web/src/components/design-system/{pill,pill-button,page-content,panel,section-heading,toolbar,texture-canvas}` call sites to use the new primitives. `pill` alone has ~30 importers, making this the single highest-risk migration task in the project. Also migrates `web/src/components/ui/empty.tsx` (~10 importers).

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST create `PageHeader`, `Pills`, `SearchInput`, `Empty`, `Section`, `Toolbar` primitives in `packages/ui/src/components/` and export them from `packages/ui/src/index.ts`.
- MUST follow the visual + prop contracts defined in `docs/design/web-inspiration/src/primitives.jsx`.
- MUST map existing `design-system/pill` tone variants (neutral, amber, green, violet, danger) to the `Pills` semantic tint system defined in DESIGN.md §4.
- MUST rewrite all ~30 importers of `@/components/design-system/pill` to use `@agh/ui` `Pills` in the same PR.
- MUST rewrite importers of `@/components/design-system/{pill-button,page-content,panel,section-heading,toolbar}` to compose `Pills`/`Button`/`PageHeader`/`Card`/`Toolbar` from `@agh/ui`.
- MUST rewrite importers of `@/components/ui/empty` to use `@agh/ui` `Empty`.
- MUST DELETE the five migrated files from `web/src/components/design-system/` and the `empty.tsx` file from `web/src/components/ui/`.
- MUST NOT migrate `texture-canvas.tsx` — delete it outright; DESIGN.md forbids decorative texture except the marketing hero mesh.
- MUST add stories for each new primitive covering variants, empty/populated states, disabled and active states.
- SHOULD batch the ~30 pill rewrite commits by domain (tasks/, session/, network/, etc.) for reviewability — one final PR, multiple clean commits.
</requirements>

## Subtasks

- [x] 6.1 Implement `PageHeader`, `Pills`, `SearchInput`, `Empty`, `Section`, `Toolbar` with stories.
- [x] 6.2 Audit every `design-system/pill` importer and map its tone prop to the new `Pills` variants.
- [x] 6.3 Rewrite `design-system/{pill,pill-button,page-content,panel,section-heading,toolbar}` importers in `web/src/**`.
- [x] 6.4 Rewrite `ui/empty` importers to use `@agh/ui` `Empty`.
- [x] 6.5 Delete the migrated source files (including `texture-canvas.tsx` without replacement).
- [x] 6.6 Run `make verify`; fix regressions; confirm every screen using the legacy primitives still renders.

## Implementation Details

Primitive shapes are in TechSpec "Core Interfaces" and DESIGN.md §4. The mock at `docs/design/web-inspiration/src/primitives.jsx` defines the exact composition of `PageHeader`, `Pills` (segmented toggle), `SearchInput`, `Empty`, `Section`, `Metric`.

Tone mapping reference (design-system → `@agh/ui`):

- `tone="neutral"` → `Pills` default (border + muted text)
- `tone="amber"` → `Pills variant="warning"`
- `tone="green"` → `Pills variant="success"`
- `tone="violet"` → `Pills variant="info"`
- `tone="danger"` → `Pills variant="danger"`
- `tone="accent"` (if used) → `Pills variant="accent"` (new)

### Relevant Files

- `web/src/components/design-system/pill.tsx`, `pill-button.tsx`, `page-content.tsx`, `panel.tsx`, `section-heading.tsx`, `toolbar.tsx`, `texture-canvas.tsx` — migration sources.
- `web/src/components/ui/empty.tsx` — migration source.
- `packages/ui/src/components/` — destination for six new primitives.
- `packages/ui/src/index.ts` — add exports.
- `docs/design/web-inspiration/src/primitives.jsx` — reference implementation.
- Grep `@/components/design-system/pill` + all other migrated paths for call sites.

### Dependent Files

- ~30 importers of `design-system/pill` across every domain system.
- `design-system-showcase.tsx` (rewritten in task 15) no longer imports from the migrated primitives.
- Task 07 migrates the remaining `design-system/` residents; task 15 deletes the folder.

### Related ADRs

- [ADR-001: Consolidate UI primitives into @agh/ui](adrs/adr-001.md)
- [ADR-002: Greenfield migration](adrs/adr-002.md)

## Deliverables

- Six new primitives with stories.
- All `design-system/{pill,pill-button,page-content,panel,section-heading,toolbar,texture-canvas}` files deleted and importers rewritten.
- `web/src/components/ui/empty.tsx` deleted and importers rewritten.
- Unit tests with 80%+ coverage for each primitive **(REQUIRED)**.
- Storybook interaction tests for Pills selection + SearchInput typing **(REQUIRED)**.

## Tests

- Unit tests:
  - [ ] `Pills` with `variant="success"` renders the success tint token as background.
  - [ ] `Pills` in segmented (toggle) mode fires `onChange` with the selected value.
  - [ ] `PageHeader` renders title + optional icon + count badge + right-side actions in the mock-specified order.
  - [ ] `SearchInput` fires `onChange` on keystroke and renders the kbd hint.
  - [ ] `Empty` renders centered icon + title + description + optional action.
  - [ ] `Section` renders label + optional right slot + children.
  - [ ] `Toolbar` composes SearchInput + actions horizontally and wraps on narrow viewports.
- Integration tests:
  - [ ] Storybook `play()` clicks each segment of `Pills` in toggle mode and asserts active state flip.
  - [ ] A domain component previously using `design-system/pill` renders unchanged after the rewrite (check Tasks list row tone variants).
- Test coverage target: >=80%
- All tests must pass

## Success Criteria

- All tests passing
- Test coverage >=80%
- `rg "@/components/design-system/(pill|pill-button|page-content|panel|section-heading|toolbar|texture-canvas)" web/src` returns zero matches.
- `rg "@/components/ui/empty" web/src` returns zero matches.
- Seven files deleted from `web/src/components/design-system/` (leaving metric, metric-strip, status-dot, connection-indicator, design-system-showcase for task 07/15).
- `web/src/components/ui/empty.tsx` deleted.
- `make verify` passes.
