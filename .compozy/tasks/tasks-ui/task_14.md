---
status: completed
title: List, kanban, empty-state, and create modal
type: frontend
complexity: high
dependencies:
  - task_13
---

# Task 14: List, kanban, empty-state, and create modal

## Overview

Implement the main browsing and creation surfaces for the Tasks area: split-view list, kanban view, empty state, and create modal. This task delivers the core operator entry experience, including search/filter state, draft creation, and publish-aware create flows that sit on top of the new tasks system.

<critical>
- ALWAYS READ `_techspec.md`, ADRs, `task_12.md`, `task_13.md`, and the relevant `analysis/*.md` files before building the main tasks surface
- REFERENCE TECHSPEC sections "System Architecture", "API Endpoints", "Testing Approach", and the screen analysis for list, kanban, empty state, and create modal
- FOCUS ON "WHAT" — implement the primary tasks browsing and creation workflows, not unrelated dashboard or live-detail views
- MINIMIZE CODE — reuse the shared tasks system and route hook instead of adding per-component fetch state
- TESTS REQUIRED — list states, kanban grouping, create modal behavior, and draft/publish flows all need coverage
- GREENFIELD: list e kanban precisam nascer com o read model rico; nao degrade a UX para um CRUD basico que depois precise ser refeito inteiro
</critical>

<requirements>
- MUST implement the split-view tasks list with search, filters, selection, and an enriched card presentation
- MUST implement the kanban view using the enriched task summaries rather than a client-only reconstruction of lifecycle state
- MUST implement the empty-state experience and task templates as first-run UX for the tasks area
- MUST implement the create modal with support for draft creation, publication-aware flows, and first-class task semantics such as priority and attempts
- MUST keep components presentational and route/system hooks responsible for orchestration
- SHOULD preserve stable layout behavior when switching between list, kanban, empty, and create states
</requirements>

## Subtasks
- [x] 14.1 Build the base tasks page orchestration for list and kanban mode switching
- [x] 14.2 Implement split-view list panels and enriched task-card rendering
- [x] 14.3 Implement kanban grouping, card actions, and empty-column behavior
- [x] 14.4 Implement the empty state and create modal flows, including draft-aware create behavior
- [x] 14.5 Add route and component tests for loading, error, empty, draft, and populated states

## Implementation Details

See TechSpec sections "System Architecture", "Testing Approach", and the analysis docs for list/split view, kanban, empty state, and create modal. This task should use the tasks system from task_13 and keep route-level state in `use-tasks-page.ts`.

### Relevant Files
- `web/src/routes/_app/tasks.tsx` — base tasks area route that will render list, kanban, and empty-state surfaces
- `web/src/hooks/routes/use-tasks-page.ts` — route-level orchestration for mode switching, search, selection, and create-modal state
- `web/src/systems/tasks/components/` — new list, kanban, empty-state, and create-modal components introduced by this task
- `.compozy/tasks/tasks-ui/analysis/analysis_list-split-view.md` — split-view data and behavior requirements
- `.compozy/tasks/tasks-ui/analysis/analysis_kanban-view.md` — kanban-specific grouping and card requirements
- `.compozy/tasks/tasks-ui/analysis/analysis_empty-state.md` — template and first-run UX expectations
- `.compozy/tasks/tasks-ui/analysis/analysis_create-modal.md` — create-modal requirements, draft semantics, and gaps
- `docs/design/paper/tasks/` — local Paper exports for the corresponding screens

### Dependent Files
- `web/src/routes/_app/-tasks.test.tsx` — route-level coverage for list, kanban, empty, and create states
- `web/src/systems/tasks/**/*.test.tsx` — component and hook tests for the main tasks browsing/creation surfaces
- `web/e2e/tasks.spec.ts` or `web/e2e/tasks-*.spec.ts` — future browser coverage in task_19 will exercise these flows
- `web/src/hooks/routes/use-task-detail-page.ts` — task_15 will share selection and deep-link behavior with this page flow

### Related ADRs
- [ADR-001: First-Class Tasks Area in the Main App Shell](adrs/adr-001.md) — Requires Tasks to behave as a primary operator surface
- [ADR-002: Expand the Task Domain for Paper-Parity Semantics](adrs/adr-002.md) — Create flows must honor first-class priority, draft, and attempt semantics

## Deliverables
- Split-view list and kanban experiences for Tasks
- Empty-state UX and create modal with draft-aware behavior
- Route and component tests with >=80% coverage for the main tasks browsing/creation surfaces **(REQUIRED)**
- Stable route-hook orchestration for search, selection, and creation state **(REQUIRED)**
- No direct component-level data fetching outside the shared tasks system

## Tests
- Unit tests:
  - [ ] List and kanban mode switching updates the expected page state without layout regressions or selection loss
  - [ ] Split-view cards render enriched task data such as counts, activity, active-run indicators, and first-class draft or approval semantics
  - [ ] Search, filtering, and sorting controls update the visible list set without breaking split-view selection behavior
  - [ ] Kanban grouping maps task statuses into the expected columns, preserves card actions, and handles empty-column states
  - [ ] Create modal supports draft and publish-aware flows with validation for title, priority, max-attempts, and approval-related fields
  - [ ] Empty-state templates and CTAs render correctly and route into create flows when no tasks exist
- Integration tests:
  - [ ] The base tasks route handles loading, error, empty, and populated states correctly across both list and kanban modes
  - [ ] Creating a task invalidates and refreshes the relevant list or kanban queries while preserving route-shell stability
  - [ ] Draft creation and publication flows update page state, selection, and visible task grouping coherently
  - [ ] Switching between list and kanban preserves compatible filters, search state, and create-modal behavior across route refreshes
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80% for modified route, hook, and component files
- The Tasks area has a usable main experience for browsing and creating tasks
- Operators can move between list, kanban, empty, and create flows without missing core semantics
