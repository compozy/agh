---
status: pending
title: Detail timeline and run detail routes
type: frontend
complexity: high
dependencies:
  - task_13
---

# Task 15: Detail timeline and run detail routes

## Overview

Implement the task-detail and run-detail deep-link experiences on top of the task-native live and run-detail APIs. This task should make it possible to inspect one task, follow its timeline, jump to a run detail view, and drill into the linked session context without making session SSE the primary data model.

<critical>
- ALWAYS READ `_techspec.md`, ADRs, `task_12.md`, `task_13.md`, and the detail/run analysis docs before building these routes
- REFERENCE TECHSPEC sections "System Architecture", "Core Interfaces", "API Endpoints", and "Testing Approach"
- FOCUS ON "WHAT" — implement task detail and run detail UX around task-native reads, not generic session views
- MINIMIZE CODE — reuse the dedicated task detail/run hooks and shared tasks system instead of bespoke route fetches
- TESTS REQUIRED — deep-link routing, timeline rendering, run-detail actions, and fallback states all need coverage
- GREENFIELD: task detail e run detail precisam depender primeiro da superficie task-native; session transcript entra como drill-down, nao como join principal
</critical>

<requirements>
- MUST implement the task detail route with task-native timeline, dependency, child-task, and activity presentation
- MUST implement the run detail route with run summary, timing/metrics presentation, and linked-session drill-down behavior
- MUST keep route state and fetching in the dedicated detail/run route hooks
- MUST handle loading, empty, disconnected-live, and error states gracefully for both routes
- SHOULD expose clear navigation between the main tasks surface, task detail, and run detail deep links
</requirements>

## Subtasks
- [ ] 15.1 Build task-detail route orchestration over the task detail and timeline reads
- [ ] 15.2 Implement timeline, dependency, child-task, and linked-run UI for the task-detail screen
- [ ] 15.3 Build run-detail route orchestration over the task run-detail read
- [ ] 15.4 Implement run-detail panels, linked-session drill-down affordances, and fallback states
- [ ] 15.5 Add route and component tests for deep links, live states, and run-detail behavior

## Implementation Details

See TechSpec sections "Core Interfaces", "API Endpoints", and the analysis docs for task detail and run detail. These routes should consume the dedicated task-native reads from the tasks system and keep session-related behavior secondary.

### Relevant Files
- `web/src/routes/_app/tasks.$id.tsx` — task-detail deep-link route
- `web/src/routes/_app/tasks.$id.runs.$runId.tsx` — run-detail deep-link route
- `web/src/hooks/routes/use-task-detail-page.ts` — detail route orchestration for task-native live reads
- `web/src/hooks/routes/use-task-run-page.ts` — run-detail orchestration for task-native run-detail reads
- `web/src/systems/tasks/components/` — new timeline, detail, and run-detail components introduced by this task
- `.compozy/tasks/tasks-ui/analysis/analysis_detail-events-sse.md` — task detail/timeline expectations
- `.compozy/tasks/tasks-ui/analysis/analysis_run-detail.md` — run-detail expectations
- `docs/design/paper/tasks/` — local Paper exports for detail and run detail

### Dependent Files
- `web/src/routes/_app/-tasks.$id.test.tsx` — detail route coverage
- `web/src/routes/_app/-tasks.$id.runs.$runId.test.tsx` — run-detail route coverage
- `web/src/systems/tasks/**/*.test.tsx` — hook/component coverage for timeline and run detail
- `web/e2e/tasks.spec.ts` or `web/e2e/tasks-*.spec.ts` — browser QA in task_19 will exercise these routes
- `web/src/systems/session/` — linked-session drill-down remains a dependent destination, not the source of truth for these routes

### Related ADRs
- [ADR-003: Add Dedicated Task Live Surfaces Instead of Client-Side Stitching](adrs/adr-003.md) — Detail and run-detail routes must depend on task-native timeline and run-detail APIs

## Deliverables
- Task detail route with task-native timeline and related task state
- Run detail route with run summary and linked-session drill-down
- Route and component tests with >=80% coverage for detail and run-detail behavior **(REQUIRED)**
- Stable deep-link navigation between main tasks, task detail, and run detail **(REQUIRED)**
- Graceful live/fallback states for both routes **(REQUIRED)**

## Tests
- Unit tests:
  - [ ] Task-detail route hook resolves task ID, selection state, active tab, and timeline query behavior correctly
  - [ ] Timeline components render task events, linked run state, cursor or pagination changes, and fallback states correctly
  - [ ] Run-detail route hook resolves run ID, linked task context, and drill-down actions correctly from deep links
  - [ ] Run-detail UI renders timing, metrics, tool or token summaries, and loading or error states without relying on raw session data
  - [ ] Navigation affordances back to task detail or list preserve search params and selection context coherently
- Integration tests:
  - [ ] Direct navigation to a task-detail route loads the expected task-native content and timeline scaffolding
  - [ ] Timeline refresh, pagination, or live-update fallback behavior does not duplicate rows or lose linked run context
  - [ ] Direct navigation to a run-detail route loads the expected run summary and linked-session affordances
  - [ ] Navigation between list, detail, and run-detail routes preserves selection and deep-link state coherently
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80% for modified detail/run route files
- Operators can inspect a task and one of its runs through stable deep links
- The frontend uses task-native timeline and run-detail APIs as the primary model for these screens
