---
status: completed
title: Rewrite Knowledge domain (list + detail)
type: frontend
complexity: medium
dependencies:
  - task_13
  - task_14
---

# Task 26: Rewrite Knowledge domain (list + detail)

<critical name="redesign-execution">
**Agent:** Run implementation with the designer agent defined in `.claude/agents/designer.md` — **execution mode only** (e.g. `mode: execute`, `execution mode`, "ship it"). **Do not** use plan mode (no `questions_v2`, no brainstorm-only passes) for these tasks.

**Mandatory skills:** `agh-design`, `design-taste-frontend`, `minimalist-ui` — activate before writing or changing UI.

**Design system:** `DESIGN.md` (repo root) is the authoritative design-system spec; tokens and rules there override informal styling.
</critical>

## Overview

Rewrite `web/src/systems/knowledge/**` as a split-pane view of markdown knowledge docs: the left list groups entries by scope (global / workspace) with search + scope filter; the right detail preview renders the markdown body in the mono `CodeBlock` primitive alongside a metadata section and destructive actions. Domain adapters, hooks, types, and query wiring stay as-is — only the visual layer is rewritten on top of `@agh/ui` primitives per ADR-001.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST rewrite `knowledge-list-panel.tsx` and `knowledge-detail-panel.tsx` as compositions over `@agh/ui` `SplitPane`, `PageHeader`, `SearchInput`, `Pills`, `MonoBadge`, `Section`, `CodeBlock`, `Empty`, `Button`, and `Dialog` (for edit/delete confirm).
- MUST replace the local `PillButton` scope tabs (`all` / `global` / `workspace`) with `@agh/ui` `Pills` wired into the existing `useKnowledgePage` state.
- MUST render the markdown preview inside the shared `CodeBlock` primitive (mono, read-only) — no bespoke `<pre>` styling.
- MUST render scope, type, agent, and modified-at metadata inside a `Section` with `MonoBadge` labels instead of the current hand-rolled metadata table.
- MUST preserve all existing behaviors from `useKnowledgePage` — scope filter, search, selection, delete mutation, dream-status meta, memory count.
- MUST NOT import from `@/components/ui/*` or `@/components/design-system/*` (both folders are deleted in Phase 2).
- MUST keep `web/src/routes/_app/knowledge.tsx` as a thin route component that wires `useKnowledgePage` into the new composition; route path `/knowledge` is unchanged.
- SHOULD consolidate scope derivation and badge tone mapping into a small `lib/` helper so the component layer stays presentational.
</requirements>

## Subtasks

- [x] 26.1 Audit `web/src/systems/knowledge/components/**`, `routes/_app/knowledge.tsx`, and the `useKnowledgePage` hook to catalogue every prop and behavior that must survive the rewrite.
- [x] 26.2 Rewrite `knowledge-list-panel.tsx` on `SplitPane.list` using `SearchInput`, `Pills`, grouped list rows with `MonoBadge` for type + scope.
- [x] 26.3 Rewrite `knowledge-detail-panel.tsx` on `SplitPane.detail` with `PageHeader`, `Section`, `CodeBlock` preview, metadata rows, and `Button` actions; gate delete behind a `Dialog` confirmation.
- [x] 26.4 Rewrite `routes/_app/knowledge.tsx` to compose `SplitPane` + the new panels; remove the `WorkspacePageShell` wrapper once header slots move into `PageHeader`.
- [x] 26.5 Update or add Storybook stories for list, detail, loading, error, empty-selection, and scope-filtered states.
- [x] 26.6 Regenerate Playwright visual snapshot baselines for `/knowledge` (empty, populated, detail-open, delete-dialog).
- [x] 26.7 Run `make web-lint`, `make web-typecheck`, `make web-test`, and `make verify`.

## Implementation Details

Follow TechSpec "Impact Analysis" row `web/src/systems/knowledge/**` — visual-only rewrite, data flow untouched. The mock `KnowledgePage` in the archived `docs/design/web-inspiration` reference is the visual target; primitive composition matches the split-pane pattern established in task 13/14 and mirrored in Network/Automation/Bridges tasks.

### Relevant Files

- `web/src/systems/knowledge/components/knowledge-list-panel.tsx` — rewrite target.
- `web/src/systems/knowledge/components/knowledge-detail-panel.tsx` — rewrite target.
- `web/src/systems/knowledge/components/stories/*.stories.tsx` — update for new primitives.
- `web/src/routes/_app/knowledge.tsx` — thin route composition rewrite.
- `web/src/hooks/routes/use-knowledge-page.ts` — unchanged consumer of the new panels.
- `packages/ui/src/components/{split-pane,page-header,pills,search-input,mono-badge,section,code-block,empty,dialog}.tsx` — primitives consumed.
- **Design references** (read-only, do not edit):
  - `DESIGN.md §4` — markdown preview + metadata section spec.
  - `docs/design/web-inspiration/src/pages-core.jsx` — `KnowledgePage` split-pane composition.

### Dependent Files

- `web/e2e/__snapshots__/` — new baselines for `/knowledge` route states.
- `web/src/routes/_app/-knowledge.test.tsx` — route test updated to assert new testids.

### Related ADRs

- [ADR-001: Consolidate UI primitives into @agh/ui](adrs/adr-001.md)
- [ADR-002: Greenfield migration](adrs/adr-002.md)
- [ADR-004: Phased rollout](adrs/adr-004.md)
- [ADR-005: Visual parity via Playwright snapshots](adrs/adr-005.md)

## Deliverables

- Rewritten `knowledge-list-panel.tsx` and `knowledge-detail-panel.tsx` composed from `@agh/ui` primitives only.
- Rewritten `routes/_app/knowledge.tsx` route.
- Storybook stories covering list / detail / loading / error / empty / scope-filtered.
- Playwright snapshot baselines for `/knowledge` empty, populated, detail-open, and delete-dialog states **(REQUIRED)**.
- Unit tests with 80%+ coverage **(REQUIRED)**.
- Storybook interaction tests for search + scope filter + select-row + delete confirmation **(REQUIRED)**.

## Tests

- Unit tests:
  - [x] `KnowledgeListPanel` renders grouped sections ordered GLOBAL before WORKSPACE.
  - [x] `SearchInput` filters rows by name, description, and type (case-insensitive).
  - [x] `Pills` scope filter (`all` / `global` / `workspace`) updates selection and calls `setActiveTab`.
  - [x] Selecting a row emits `onSelectMemory` with the clicked filename and renders the `3px` accent indicator.
  - [x] `KnowledgeDetailPanel` renders `Empty` state when no memory is selected.
  - [x] `KnowledgeDetailPanel` renders the markdown preview inside `CodeBlock` and `MonoBadge` chips for type + scope.
  - [x] Delete button opens the `Dialog` confirm and calls `onDelete(filename)` on confirm; disabled while `isDeletePending`.
- Integration tests:
  - [x] Storybook `play()` types a query into `SearchInput` and asserts filtered rows render.
  - [x] Storybook `play()` selects a row and asserts the detail panel renders the preview, metadata, and enabled delete button.
  - [x] Playwright visual snapshot match for `/knowledge` empty, populated, detail-open, and delete-dialog states.
- Test coverage target: >=80%
- All tests must pass

## Success Criteria

- All tests passing
- Test coverage >=80%
- `web/src/systems/knowledge/**` contains zero imports from `@/components/ui/*` or `@/components/design-system/*`.
- `/knowledge` renders only through `@agh/ui` primitives + domain hooks.
- Playwright baseline snapshots committed for all four `/knowledge` states.
- `make verify` passes.
