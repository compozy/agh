---
status: pending
title: Add Sidebar + SplitPane primitives to @agh/ui
type: frontend
complexity: high
dependencies:
  - task_01
---

# Task 05: Add Sidebar + SplitPane primitives to @agh/ui

<critical name="redesign-execution">
**Agent:** Run implementation with the designer agent defined in `.claude/agents/designer.md` — **execution mode only** (e.g. `mode: execute`, `execution mode`, "ship it"). **Do not** use plan mode (no `questions_v2`, no brainstorm-only passes) for these tasks.

**Mandatory skills:** `agh-design`, `design-taste-frontend`, `minimalist-ui` — activate before writing or changing UI.

**Design system:** `DESIGN.md` (repo root) is the authoritative design-system spec; tokens and rules there override informal styling.
</critical>

## Overview

Introduce two new structural primitives to `@agh/ui`: `Sidebar` (workspace rail + panel shell with slots for rail, header, nav, and footer) and `SplitPane` (fixed-width list column + flex detail column). These replace the generic shadcn `sidebar.tsx` currently in `web/src/components/ui/` and become the foundation for the whole app layout. Derived from `docs/design/web-inspiration/src/sidebar.jsx` and `primitives.jsx`.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST create `packages/ui/src/components/sidebar.tsx` implementing the `Sidebar` shell with slots `rail`, `header`, `nav`, `footer`, and props `collapsed` + `onCollapse`.
- MUST create `packages/ui/src/components/split-pane.tsx` implementing `SplitPane` with props `list`, `detail`, `listWidth` (default 340), `detailEmpty`.
- MUST export both from `packages/ui/src/index.ts`.
- MUST write stories that demonstrate: collapsed vs expanded Sidebar, SplitPane with selected + empty detail, SplitPane on narrow viewports (collapses to single column with back navigation).
- MUST use `motion` for the sidebar collapse width transition (200ms) and for the detail panel enter/exit.
- MUST NOT include any AGH-specific content in the shell (no workspace lists, no agent trees) — those are slot children injected by the app.
- MUST delete `web/src/components/ui/sidebar.tsx` (the shadcn one) and rewrite its importers to either compose `@agh/ui` Sidebar directly or move to `SplitPane` where appropriate.
- SHOULD keep the Sidebar shell responsive — rail (40–44px) is always visible; panel collapses on narrow viewports.
</requirements>

## Subtasks

- [ ] 5.1 Implement `Sidebar` with slotted rail/header/nav/footer.
- [ ] 5.2 Implement `SplitPane` with list/detail columns, empty state fallback, and responsive collapse.
- [ ] 5.3 Export both + write stories covering expanded/collapsed, selected/empty, narrow viewport.
- [ ] 5.4 Delete `web/src/components/ui/sidebar.tsx` and rewrite the ~5 importers to the new `@agh/ui` export.
- [ ] 5.5 Verify `make verify` passes.

## Implementation Details

See TechSpec "Core Interfaces" for `SidebarProps` and `SplitPaneProps` signatures. DESIGN.md §4 "Sidebar (Operator UI)" codifies the visual expectations. The mock in `docs/design/web-inspiration/src/sidebar.jsx` shows the slot composition.

### Relevant Files

- `packages/ui/src/components/sidebar.tsx` — new.
- `packages/ui/src/components/split-pane.tsx` — new.
- `packages/ui/src/index.ts` — add exports.
- `packages/ui/src/components/stories/sidebar.stories.tsx` + `split-pane.stories.tsx` — new.
- `web/src/components/ui/sidebar.tsx` — to delete.
- `docs/design/web-inspiration/src/sidebar.jsx` — reference composition.
- `docs/design/web-inspiration/src/primitives.jsx` — `Section` / `PageHeader` patterns.

### Dependent Files

- Task 08 (folder close-out) depends on `web/src/components/ui/sidebar.tsx` being gone.
- Task 13 (app-sidebar rewrite) consumes `Sidebar`.
- Task 14 (root layout) consumes `SplitPane`.
- Tasks 17, 20, 23–27 (domain screens with list + detail) consume `SplitPane`.

### Related ADRs

- [ADR-001: Consolidate UI primitives into @agh/ui](adrs/adr-001.md)
- [ADR-003: Adopt motion for UI animations](adrs/adr-003.md)

## Deliverables

- `Sidebar` primitive with slots + collapse animation.
- `SplitPane` primitive with list/detail layout + motion-driven detail entry.
- Stories covering the required states.
- `web/src/components/ui/sidebar.tsx` deleted and its importers rewritten.
- Unit tests with 80%+ coverage **(REQUIRED)**.
- Storybook interaction tests for collapse + selection **(REQUIRED)**.

## Tests

- Unit tests:
  - [ ] `Sidebar` renders all four slots (rail/header/nav/footer) when provided.
  - [ ] `Sidebar` calls `onCollapse(true)` when the collapse control is activated.
  - [ ] `Sidebar` transitions width via motion when `collapsed` changes.
  - [ ] `SplitPane` renders the `list` column at the configured width and the `detail` column flex-1.
  - [ ] `SplitPane` renders `detailEmpty` when `detail` is `null`.
  - [ ] Under `prefers-reduced-motion: reduce`, sidebar width change is instant (no animated transition).
- Integration tests:
  - [ ] Storybook `play()` clicks collapse, asserts `aria-expanded` flips, and verifies rail stays visible.
  - [ ] Storybook `play()` for `SplitPane` selects a list item and asserts the detail panel renders.
- Test coverage target: >=80%
- All tests must pass

## Success Criteria

- All tests passing
- Test coverage >=80%
- `Sidebar` and `SplitPane` exported from `packages/ui/src/index.ts`.
- `web/src/components/ui/sidebar.tsx` deleted; no remaining imports of it.
- `make verify` passes.
