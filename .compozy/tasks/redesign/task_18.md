---
status: done
title: Rewrite Tasks domain Kanban, Dashboard, Inbox views
type: frontend
complexity: high
dependencies:
  - task_17
---

# Task 18: Rewrite Tasks domain Kanban, Dashboard, Inbox views

<critical name="redesign-execution">
**Agent:** Run implementation with the designer agent defined in `.claude/agents/designer.md` — **execution mode only** (e.g. `mode: execute`, `execution mode`, "ship it"). **Do not** use plan mode (no `questions_v2`, no brainstorm-only passes) for these tasks.

**Mandatory skills:** `agh-design`, `design-taste-frontend`, `minimalist-ui` — activate before writing or changing UI.

**Design system:** `DESIGN.md` (repo root) is the authoritative design-system spec; tokens and rules there override informal styling.
</critical>

## Overview

Rewrite the three remaining Tasks views — Kanban (4 columns), Dashboard (metrics + charts), and Inbox (approval flow) — against `@agh/ui`. All three reuse the shared `tasks-list-row.tsx` primitive introduced in task 17 and share the `tasks-page-shell.tsx` view switcher. This completes the visual surface users see when landing on `/tasks` regardless of which view they last selected.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST rewrite `tasks-kanban-board.tsx` as four `@agh/ui` `Section`-wrapped columns labeled Pending, Running, Done, Failed, each rendering the shared list-row component from task 17.
- MUST rewrite `tasks-dashboard-view.tsx`, `tasks-dashboard-cards.tsx`, `tasks-dashboard-active-runs.tsx`, `tasks-dashboard-status-breakdown.tsx`, and `tasks-dashboard-queue-health.tsx` with a 4-metric top row (`Metric` x4), Active Runs + Status Breakdown cards (`Section`), and a Queue Health 24h chart card.
- MUST rewrite `tasks-inbox-view.tsx`, `tasks-inbox-item.tsx`, and `tasks-inbox-lane-tabs.tsx` as a 5-item list with per-row Approve / Edit / Reject actions (`@agh/ui` `Button`) and an unread dot driven by `item.unread`.
- MUST reuse the shared `tasks-list-row.tsx` from task 17 inside Kanban columns and Inbox rows where the row's metadata layout matches; differences (Approve/Edit/Reject affordances) go in the Inbox-specific wrapper, not by forking the row.
- MUST preserve the existing lane-tab state (`tasks-inbox-lane-tabs.tsx`) and query-hook wiring; only the presentational layer changes.
- MUST NOT import from `web/src/components/ui/**` or `web/src/components/design-system/**`.
- MUST keep existing Vitest tests under `web/src/systems/tasks/components/*.test.tsx` green unless the component's props change in this PR.
- SHOULD reuse the Queue Health chart library already present in `tasks-dashboard-queue-health.tsx` (Recharts or equivalent); only the wrapping `Section` and chart theming tokens change.
</requirements>

## Subtasks

- [x] 18.1 Rewrite `tasks-kanban-board.tsx` on `Section` columns + shared list-row; wire the drag/reorder behavior that exists today unchanged.
- [x] 18.2 Rewrite the five `tasks-dashboard-*.tsx` files on `Metric` + `Section`, updating chart tokens to `--color-accent` and `--color-accent-tint-strong`.
- [x] 18.3 Rewrite `tasks-inbox-view.tsx` + `tasks-inbox-item.tsx` + `tasks-inbox-lane-tabs.tsx`; ensure Approve/Edit/Reject buttons use the correct `Button` variants (primary / ghost / danger) and an unread dot renders as a `StatusDot` with `tone="accent"` when `item.unread`.
- [x] 18.4 Rewrite Storybook stories for all three views covering empty, populated, loading, error, and view-switcher transitions from List to each view.
- [x] 18.5 Update Vitest tests only where component props or DOM structure changed; verify untouched tests still pass.
- [x] 18.6 Run `make verify` and `pnpm test:visual`; commit Playwright baselines for Kanban, Dashboard, and Inbox routes.

## Implementation Details

See TechSpec §"Impact Analysis" for the Tasks domain rewrite scope and DESIGN.md §4 for primitive visuals. The Kanban / Dashboard / Inbox layouts in the mock live at `docs/design/web-inspiration/src/pages-core.jsx` under `TasksKanbanView`, `TasksDashboardView`, and `TasksInboxView`. The task 17 shared row primitive is the reuse anchor — do not duplicate row layout code.

Inbox action-to-button-variant mapping:

- Approve → `Button variant="primary"`
- Edit → `Button variant="ghost"`
- Reject → `Button variant="danger"` (outline when inactive)

Dashboard metric set (top row, left to right):

- Active runs (count, delta vs. prior 24h)
- Success rate (percent, 24h window)
- Average duration (ms/s, rolling 24h)
- Queue depth (count, latest sample)

### Relevant Files

- `web/src/systems/tasks/components/tasks-kanban-board.tsx` — rewrite target.
- `web/src/systems/tasks/components/tasks-dashboard-view.tsx` — rewrite target (dashboard shell).
- `web/src/systems/tasks/components/tasks-dashboard-cards.tsx` — rewrite on 4 `Metric` primitives.
- `web/src/systems/tasks/components/tasks-dashboard-active-runs.tsx` — rewrite on `Section` + shared list-row.
- `web/src/systems/tasks/components/tasks-dashboard-status-breakdown.tsx` — rewrite on `Section` + `Pills` counts.
- `web/src/systems/tasks/components/tasks-dashboard-queue-health.tsx` — rewrite on `Section` + existing chart lib.
- `web/src/systems/tasks/components/tasks-inbox-view.tsx` — rewrite target.
- `web/src/systems/tasks/components/tasks-inbox-item.tsx` — rewrite on shared list-row + action `Button`s.
- `web/src/systems/tasks/components/tasks-inbox-lane-tabs.tsx` — rewrite on `@agh/ui` `Tabs`.
- `web/src/systems/tasks/components/tasks-list-row.tsx` — consumed from task 17; do not modify without cross-PR coordination.
- `web/src/routes/_app/tasks.tsx` — view-switcher already rendered in task 17; confirm Kanban/Dashboard/Inbox sub-routes mount correctly.
- **Design references** (read-only, do not edit):
  - `DESIGN.md §4 + §5` — column layouts, metric dashboard card patterns.
  - `docs/design/web-inspiration/src/pages-core.jsx` — `TasksPage` kanban (4 columns), dashboard (metric row + charts), inbox (approval flow).
  - `docs/design/web-inspiration/src/primitives.jsx` — shared `Pills` view-switcher + `Metric` tiles.

### Dependent Files

- `web/src/systems/tasks/hooks/**` — consumed as-is.
- `web/src/integrations/tanstack-query/**` MSW fixtures — consumed as-is.
- Task 19 (forms + run detail) is independent of this task.

### Related ADRs

- [ADR-001: Consolidate UI primitives into @agh/ui](adrs/adr-001.md)
- [ADR-004: Phased rollout](adrs/adr-004.md) — Phase 3, step 2.
- [ADR-005: Visual parity via Playwright snapshots](adrs/adr-005.md)

## Deliverables

- Rewritten Kanban, Dashboard (5 files), and Inbox (3 files) components.
- Storybook stories for each covering empty, populated, loading, and error states.
- Playwright visual baselines for the three views.
- Unit tests with 80%+ coverage **(REQUIRED)**.
- Integration tests for Inbox approval flow and Kanban column membership **(REQUIRED)**.

## Tests

- Unit tests:
  - [ ] `tasks-kanban-board.tsx` renders exactly four columns labeled Pending, Running, Done, Failed.
  - [ ] `tasks-kanban-board.tsx` routes a task with `status === "running"` into the Running column and none of the other three.
  - [ ] `tasks-dashboard-cards.tsx` renders four `Metric` primitives with labels Active runs, Success rate, Average duration, Queue depth.
  - [ ] `tasks-dashboard-status-breakdown.tsx` renders one `Pills` chip per status with its count derived from the props (asserts the sum equals the total).
  - [ ] `tasks-dashboard-queue-health.tsx` renders a chart when given 24 hourly buckets and renders the `Empty` primitive when given zero buckets.
  - [ ] `tasks-inbox-item.tsx` renders an accent `StatusDot` when `item.unread === true` and no dot when `item.unread === false`.
  - [ ] `tasks-inbox-item.tsx` renders three action buttons; clicking Approve calls `onApprove(item.id)` once.
  - [ ] `tasks-inbox-item.tsx` shows Reject with `variant="danger"` and does not trigger navigation when clicked.
- Integration tests:
  - [ ] Storybook `play()` in `tasks-inbox-view.stories.tsx` clicks Approve on item #2, asserts the item exits the unread lane, and the query cache is updated.
  - [ ] Storybook `play()` in `tasks-kanban-board.stories.tsx` drags a Running task to Done, asserts the column membership flips and the mutation hook is called once.
  - [ ] Storybook `play()` in `tasks-dashboard-view.stories.tsx` switches the Queue Health window from 24h to 1h (if the control exists) and asserts the chart re-renders with the new bucket count.
- Test coverage target: >=80%
- All tests must pass

## Success Criteria

- All tests passing.
- Test coverage >=80% across the rewritten files.
- `rg "@/components/(ui|design-system)/" web/src/systems/tasks/components/tasks-{kanban,dashboard,inbox}*` returns zero matches.
- Kanban columns, Dashboard metric row, and Inbox list all match the mock within the 0.1% Playwright pixel-diff threshold.
- Shared list-row primitive from task 17 is imported by Kanban columns and Inbox items (no duplicate row layout).
- `make verify` passes.
