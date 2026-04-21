---
status: completed
title: Rewrite Tasks domain list + detail panel
type: frontend
complexity: high
dependencies:
  - task_13
  - task_14
---

# Task 17: Rewrite Tasks domain list + detail panel

<critical name="redesign-execution">
**Agent:** Run implementation with the designer agent defined in `.claude/agents/designer.md` — **execution mode only** (e.g. `mode: execute`, `execution mode`, "ship it"). **Do not** use plan mode (no `questions_v2`, no brainstorm-only passes) for these tasks.

**Mandatory skills:** `agh-design`, `design-taste-frontend`, `minimalist-ui` — activate before writing or changing UI.

**Design system:** `DESIGN.md` (repo root) is the authoritative design-system spec; tokens and rules there override informal styling.
</critical>

## Overview

Rewrite the Tasks domain split-pane list + detail view (routes `/tasks` and `/tasks/$id`) against `@agh/ui`. This is the first screen of Phase 3 and sets the visual vocabulary reused by the Kanban, Dashboard, Inbox, forms, and run detail tasks that follow. Domain logic (TanStack Query hooks, Zustand stores, MSW fixtures, OpenAPI types) stays untouched — only the presentational layer changes.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST compose the `/tasks` route on `@agh/ui` `SplitPane` with the list as the 340px column and the detail panel as the flex column; empty detail state uses `@agh/ui` `Empty`.
- MUST rewrite `tasks-page-shell.tsx`, `tasks-list-panel.tsx`, `tasks-empty-state.tsx`, `tasks-detail-header.tsx`, `tasks-detail-overview-panel.tsx`, `tasks-detail-preview-panel.tsx`, `tasks-detail-tabs.tsx`, `tasks-detail-children-panel.tsx`, `tasks-detail-dependencies-panel.tsx`, `tasks-detail-runs-panel.tsx`, `tasks-multi-agent-panel.tsx`, `tasks-timeline-panel.tsx`, and `task-card.tsx` against `@agh/ui` primitives (`PageHeader`, `SearchInput`, `Pills`, `StatusDot`, `MonoBadge`, `Metric`, `Section`, `CodeBlock`, `Table`).
- MUST preserve every prop signature consumed by the TanStack Query hooks in `web/src/systems/tasks/hooks/**` and the stores/adapters in `web/src/systems/tasks/{adapters,lib}/**`.
- MUST NOT import from `web/src/components/ui/**` or `web/src/components/design-system/**` (both folders are gone after task 08 / task 15).
- MUST keep the existing MSW fixtures under `web/src/integrations/tanstack-query/**` and the Vitest tests in `web/src/systems/tasks/components/*.test.tsx` green; update a test file only when its component's props change.
- MUST render the list row as: `StatusDot` (tone mapped from `task.status`) + title + `MonoBadge` id + timestamp + optional `Pills` lane badge, per the mock row.
- MUST render the detail header with `PageHeader` (title + id `MonoBadge` + status pill + right-side action slot) and the detail body as a stack of `Section` blocks.
- SHOULD extract a shared `tasks-list-row.tsx` component that task 18 (Kanban, Inbox) can reuse without duplicating status + metadata layout.
- SHOULD keep this task within 7 subtasks by deferring forms, run detail, and non-list views to tasks 18 and 19.
</requirements>

## Subtasks

- [x] 17.1 Audit every file in `web/src/systems/tasks/components/` and classify which belong to list/detail (this task) vs. Kanban/Dashboard/Inbox (task 18) vs. forms/run detail (task 19).
- [x] 17.2 Extract a shared `tasks-list-row.tsx` row primitive consumed by the list panel (and later by Kanban cards + Inbox items).
- [x] 17.3 Rewrite `tasks-page-shell.tsx` + `tasks-list-panel.tsx` on `SplitPane` + `PageHeader` + `SearchInput` + `Pills` (view switcher) + `Empty`.
- [x] 17.4 Rewrite the detail-panel family on `Section`, `Metric`, `MonoBadge`, `StatusDot`, `CodeBlock`, `Table`: `tasks-detail-header.tsx`, `tasks-detail-overview-panel.tsx`, `tasks-detail-preview-panel.tsx`, `tasks-detail-tabs.tsx`, `tasks-detail-children-panel.tsx`, `tasks-detail-dependencies-panel.tsx`, `tasks-detail-runs-panel.tsx`, `tasks-multi-agent-panel.tsx`, `tasks-timeline-panel.tsx`.
- [x] 17.5 Update the TanStack Router route files `web/src/routes/_app/tasks.tsx` and `web/src/routes/_app/tasks.$id.tsx` to render the rewritten components; keep the route loaders and search-param schemas unchanged.
- [x] 17.6 Rewrite Storybook stories covering: empty list, populated list, detail loading, detail error, detail with long title, list row per status tone (pending, running, done, failed, blocked).
- [x] 17.7 Run `make verify` and `pnpm test:visual` locally; commit new Playwright baselines for the list + detail routes.

## Implementation Details

See TechSpec §"System Architecture" and §"Impact Analysis" for the Tasks domain scope (Phase 3, 27 components). Primitive contracts are in TechSpec §"Core Interfaces" and the visual spec is DESIGN.md §4. The list + detail layout in the mock lives at `docs/design/web-inspiration/src/pages-core.jsx` under `TasksPage`.

Status-tone mapping (re-used by the shared list-row component):

- `done` → `tone="success"`
- `running` → `tone="accent"` with `pulse`
- `pending` → `tone="info"`
- `blocked` → `tone="warning"`
- `failed` → `tone="danger"`
- other → `tone="neutral"`

### Relevant Files

- `web/src/systems/tasks/components/tasks-page-shell.tsx` — rewrite target; becomes the `SplitPane` composition root.
- `web/src/systems/tasks/components/tasks-list-panel.tsx` — rewrite target; list column with `SearchInput` + `Pills` filter + rows.
- `web/src/systems/tasks/components/tasks-empty-state.tsx` — rewrite on `@agh/ui` `Empty`.
- `web/src/systems/tasks/components/task-card.tsx` — rewrite target and origin of the shared row primitive.
- `web/src/systems/tasks/components/tasks-detail-header.tsx` — rewrite on `PageHeader` + `MonoBadge` + `StatusDot`.
- `web/src/systems/tasks/components/tasks-detail-overview-panel.tsx` — rewrite on `Section` + `Metric`.
- `web/src/systems/tasks/components/tasks-detail-preview-panel.tsx` — rewrite on `Section` + `CodeBlock`.
- `web/src/systems/tasks/components/tasks-detail-tabs.tsx` — rewrite on `@agh/ui` `Tabs`.
- `web/src/systems/tasks/components/tasks-detail-children-panel.tsx`, `tasks-detail-dependencies-panel.tsx`, `tasks-detail-runs-panel.tsx`, `tasks-multi-agent-panel.tsx`, `tasks-timeline-panel.tsx` — rewrite on `Section` + `Table` + `StatusDot` + `MonoBadge`; these are the tab bodies consumed by `tasks-detail-tabs.tsx`.
- `web/src/routes/_app/tasks.tsx` — compose the `SplitPane` and wire list selection to `/tasks/$id`.
- `web/src/routes/_app/tasks.$id.tsx` — render the rewritten detail panel set.
- **Design references** (read-only, do not edit):
  - `DESIGN.md §4 + §5` — metric cards, list item visuals, split-pane grid rules.
  - `docs/design/web-inspiration/src/pages-core.jsx` — `TasksPage` list-view composition (header pills, split-pane, detail header + overview + preview + tabs).
  - `docs/design/web-inspiration/src/primitives.jsx` — `PageHeader`, `SearchInput`, `Pills`, `Empty`, `Section`, `Metric` shape contracts.
  - `docs/design/web-inspiration/src/shared.jsx` — status dot + mono label helpers applied per list row.

### Dependent Files

- `web/src/systems/tasks/hooks/**` — consumed as-is; test suites stay green.
- `web/src/systems/tasks/adapters/**` and `lib/**` — consumed as-is.
- `web/src/integrations/tanstack-query/**` MSW handlers — consumed as-is.
- Task 18 (Kanban, Dashboard, Inbox) reuses the shared `tasks-list-row.tsx` from 17.2.
- Task 19 (forms + run detail) builds on the detail header + tabs pattern established here.

### Related ADRs

- [ADR-001: Consolidate UI primitives into @agh/ui](adrs/adr-001.md) — all primitives come from `@agh/ui`.
- [ADR-002: Greenfield migration](adrs/adr-002.md) — no compat shim; old files are rewritten in place.
- [ADR-004: Phased rollout](adrs/adr-004.md) — this task is Phase 3, step 1.
- [ADR-005: Visual parity via Playwright snapshots](adrs/adr-005.md) — list + detail snapshots gate the PR.

## Deliverables

- Rewritten `tasks-page-shell.tsx`, `tasks-list-panel.tsx`, `tasks-empty-state.tsx`, `task-card.tsx`, the four `tasks-detail-*.tsx` files (header, overview, preview, tabs), and the five tab-body panels (`tasks-detail-children-panel.tsx`, `tasks-detail-dependencies-panel.tsx`, `tasks-detail-runs-panel.tsx`, `tasks-multi-agent-panel.tsx`, `tasks-timeline-panel.tsx`).
- New shared `tasks-list-row.tsx` primitive reused by this task.
- Updated `tasks.tsx` and `tasks.$id.tsx` route files.
- Storybook stories for each rewritten component covering empty, populated, loading, error, and per-status variants.
- Playwright visual baselines for `/tasks` and `/tasks/$id` (including empty detail state).
- Unit tests with 80%+ coverage **(REQUIRED)**.
- Integration tests for the list → detail selection flow **(REQUIRED)**.

## Tests

- Unit tests:
  - [ ] `tasks-list-row.tsx` renders `StatusDot` with `tone="success"` when `task.status === "done"`.
  - [ ] `tasks-list-row.tsx` renders `StatusDot` with `tone="accent"` and `pulse` when `task.status === "running"`.
  - [ ] `tasks-list-row.tsx` renders a `MonoBadge` showing the 7-char short id from `task.id`.
  - [ ] `tasks-list-panel.tsx` filters rows by title when `SearchInput` emits a non-empty query.
  - [ ] `tasks-list-panel.tsx` switches between "All / Mine / Watched" lanes via the `Pills` segmented control and updates the visible rows.
  - [ ] `tasks-empty-state.tsx` renders the `Empty` primitive with a "Create task" action that invokes the provided callback.
  - [ ] `tasks-detail-header.tsx` renders `PageHeader` with title, `MonoBadge` id, status `Pills`, and a right-side actions slot in that DOM order.
  - [ ] `tasks-detail-overview-panel.tsx` renders one `Metric` per summary field (runs, duration, owner, workspace).
  - [ ] `tasks-detail-preview-panel.tsx` wraps the task preview in `CodeBlock` with the `yaml` language when `task.kind === "yaml"`.
- Integration tests:
  - [ ] Storybook `play()` in `tasks-list-panel.stories.tsx` clicks the second row and asserts the selected state + emitted `onSelect(task.id)` callback fires once.
  - [ ] Vitest route test `web/src/routes/_app/-tasks.router.integration.test.tsx` navigates `/tasks` → clicks a row → asserts URL becomes `/tasks/$id` and the detail header renders the matching title.
  - [ ] Storybook `play()` asserts the detail `Empty` state renders when the URL has no `$id` and the right column has no selected task.
- Test coverage target: >=80%
- All tests must pass

## Success Criteria

- All tests passing.
- Test coverage >=80% across the rewritten files.
- `rg "@/components/(ui|design-system)/" web/src/systems/tasks/components` returns zero matches.
- `rg "from \"@agh/ui\"" web/src/systems/tasks/components` is the only source of visual primitives in the rewritten files.
- Playwright baselines for `/tasks` (empty + populated) and `/tasks/$id` (loading, populated, error) committed with no unintended diffs elsewhere.
- `make verify` passes.
