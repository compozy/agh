---
status: pending
title: Tasks entrypoint and route shell
type: frontend
complexity: medium
dependencies:
  - task_09
  - task_10
---

# Task 12: Tasks entrypoint and route shell

## Overview

Create the first-class Tasks area in the main app shell by wiring sidebar navigation, the base tasks route, and the deep-link route family into TanStack Router. This task should leave the web app with a stable tasks frame and route structure before screen-specific components and data flows are layered on.

<critical>
- ALWAYS READ `_techspec.md`, ADRs, and the existing app-shell route patterns before adding the tasks routes
- REFERENCE TECHSPEC sections "System Architecture", "Impact Analysis", and "Development Sequencing"
- FOCUS ON "WHAT" — establish navigation and route-shell structure, not the final task screen implementations
- MINIMIZE CODE — add one coherent tasks route family and shared shell instead of ad hoc per-screen route islands
- TESTS REQUIRED — sidebar navigation, base route activation, and deep-link route registration need coverage
- GREENFIELD: tasks entra como area primaria do app; nao esconda a feature atras de um button solto ou rota experimental
</critical>

<requirements>
- MUST add a first-class Tasks navigation entry in the app sidebar
- MUST add the `/_app/tasks` route plus the deep-link route family for task detail and run detail
- MUST keep route files thin and leave screen orchestration for later route-hook/system tasks
- MUST regenerate `web/src/routeTree.gen.ts` after the route files are added
- SHOULD preserve the existing app-shell visual language and route naming patterns used by other operator surfaces
</requirements>

## Subtasks
- [ ] 12.1 Turn the Tasks area into a real sidebar navigation target
- [ ] 12.2 Add the base tasks route and deep-link route files for detail and run detail
- [ ] 12.3 Add the shared page shell/frame that later task screens can render inside
- [ ] 12.4 Regenerate the route tree and add route-level tests for the new tasks entrypoint

## Implementation Details

See TechSpec sections "System Architecture", "Impact Analysis", and ADR-001. Follow the same app-shell and route-family conventions already used by `automation`, `network`, and `knowledge`; do not introduce a special router model for tasks.

### Relevant Files
- `web/src/components/app-sidebar.tsx` — sidebar navigation that must grow a first-class Tasks entry
- `web/src/components/app-sidebar.test.tsx` — navigation and active-state coverage
- `web/src/routes/_app/tasks.tsx` — new base tasks route
- `web/src/routes/_app/tasks.$id.tsx` — new task-detail deep-link route
- `web/src/routes/_app/tasks.$id.runs.$runId.tsx` — new run-detail deep-link route
- `web/src/routeTree.gen.ts` — generated route tree that must be refreshed after route changes

### Dependent Files
- `web/src/hooks/routes/use-tasks-page.ts` — task_13 and task_14 will compose orchestration on top of this route family
- `web/src/hooks/routes/use-task-detail-page.ts` — task_13 and task_15 will drive the detail route
- `web/src/hooks/routes/use-task-run-page.ts` — task_13 and task_15 will drive the run-detail route
- `web/src/systems/workspace/components/workspace-page-shell.tsx` — likely reference for the shared page shell structure

### Related ADRs
- [ADR-001: First-Class Tasks Area in the Main App Shell](adrs/adr-001.md) — Requires a primary Tasks route, sidebar entry, and dedicated frontend route/system structure

## Deliverables
- Sidebar Tasks entry and route-shell wiring
- Base tasks route plus task detail and run-detail route files
- Regenerated `routeTree` aligned with the new tasks route family **(REQUIRED)**
- Route and navigation tests with >=80% coverage for modified files **(REQUIRED)**
- A stable shell that later tasks can populate without reworking global navigation

## Tests
- Unit tests:
  - [ ] Sidebar renders a Tasks navigation target with the expected active-state behavior for base, detail, and run-detail routes
  - [ ] The new tasks route family is present in the generated route tree with stable path and param definitions
  - [ ] Base route components render the shared page shell without screen-specific fetch assumptions or data-shape leakage
  - [ ] Route-shell defaults for initial mode, selected task, and search-param handling remain stable when the feature first mounts
  - [ ] Loading and empty-shell placeholders render correctly before screen-specific queries resolve
- Integration tests:
  - [ ] Navigating from the sidebar reaches the tasks area successfully and keeps the app shell active-state logic correct
  - [ ] Direct navigation to task-detail and run-detail route patterns resolves the expected route components inside the shared tasks shell
  - [ ] Generated route-tree or router build checks succeed after adding the full tasks route family
  - [ ] Moving between base, detail, and run-detail routes preserves shared shell state instead of remounting into route-specific forks
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80% for modified routing/navigation files
- Tasks is a first-class area in the main app shell
- Later systems/UI tasks can implement screens without revisiting global navigation or route taxonomy
