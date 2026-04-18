---
status: completed
title: web/src/systems/tasks scaffold
type: frontend
complexity: high
dependencies:
  - task_12
---

# Task 13: web/src/systems/tasks scaffold

## Overview

Build the tasks domain scaffold in `web/src/systems/tasks/` so every later screen consumes one shared adapter, query, and hook layer. This task establishes the public barrel, generated-type wrappers, query infrastructure, and route-hook orchestration that the rest of the UI will build on.

<critical>
- ALWAYS READ `_techspec.md`, ADRs, and the existing systems patterns under `web/src/systems/`
- REFERENCE TECHSPEC sections "Core Interfaces", "System Architecture", and "Testing Approach"
- FOCUS ON "WHAT" — create the reusable tasks system layer, not final screen markup
- MINIMIZE CODE — one tasks system, one adapter layer, and one query vocabulary; avoid route-local API calls or per-screen fetch helpers
- TESTS REQUIRED — adapter errors, query keys/options, and route-hook orchestration must all be covered
- GREENFIELD: componentes de tasks nao podem chamar `apiClient` direto; tudo passa pelo sistema `web/src/systems/tasks`
</critical>

<requirements>
- MUST create a dedicated `web/src/systems/tasks/` barrel with generated-type wrappers, adapters, query keys/options, hooks, and shared helpers
- MUST keep `apiClient` usage isolated to the tasks adapter layer
- MUST add route-hook orchestration for the tasks page, task detail page, and task run page
- MUST provide reusable hooks for task lists, actions, live reads, dashboard reads, and inbox reads
- SHOULD follow the existing `automation` and `network` system patterns closely so the tasks system feels native to the repo
</requirements>

## Subtasks
- [x] 13.1 Create the tasks system barrel, domain types, and adapter error surface
- [x] 13.2 Add query keys/options for list, detail, timeline, tree, run detail, dashboard, and inbox
- [x] 13.3 Add shared task hooks for reads, mutations, and live subscriptions
- [x] 13.4 Add route hooks for the base tasks page, task detail page, and run-detail page
- [x] 13.5 Add adapter, query, and hook tests for the scaffold

## Implementation Details

See TechSpec sections "Core Interfaces", "System Architecture", and "Testing Approach". Follow the `web/src/systems/automation` and `web/src/systems/network` patterns: the tasks system should own API access, typed errors, query factories, reusable hooks, and route-facing orchestration.

### Relevant Files
- `web/src/systems/automation/index.ts` — reference public barrel structure
- `web/src/systems/automation/adapters/automation-api.ts` — reference adapter pattern for `apiClient`
- `web/src/systems/automation/lib/query-keys.ts` — reference query-key hierarchy
- `web/src/systems/automation/lib/query-options.ts` — reference colocated query options and fetchers
- `web/src/hooks/routes/use-automation-page.ts` — reference route-hook orchestration shape
- `web/src/hooks/routes/use-network-page.ts` — reference page orchestration for multi-panel operator surfaces
- `web/src/generated/agh-openapi.d.ts` — generated types that the tasks system must consume

### Dependent Files
- `web/src/systems/tasks/index.ts` — new public barrel introduced in this task
- `web/src/systems/tasks/adapters/tasks-api.ts` — new adapter layer introduced in this task
- `web/src/systems/tasks/lib/query-keys.ts` — new query-key hierarchy introduced in this task
- `web/src/systems/tasks/lib/query-options.ts` — new query options introduced in this task
- `web/src/systems/tasks/hooks/use-tasks.ts` — shared list/detail hooks introduced in this task
- `web/src/systems/tasks/hooks/use-task-actions.ts` — mutation hooks introduced in this task
- `web/src/systems/tasks/hooks/use-task-live.ts` — live-read hooks introduced in this task
- `web/src/hooks/routes/use-tasks-page.ts` — route-level orchestration introduced in this task
- `web/src/hooks/routes/use-task-detail-page.ts` — detail route orchestration introduced in this task
- `web/src/hooks/routes/use-task-run-page.ts` — run-detail route orchestration introduced in this task

### Related ADRs
- [ADR-001: First-Class Tasks Area in the Main App Shell](adrs/adr-001.md) — Requires a dedicated frontend domain system for Tasks
- [ADR-003: Add Dedicated Task Live Surfaces Instead of Client-Side Stitching](adrs/adr-003.md) — Requires reusable frontend live hooks over task-native APIs
- [ADR-004: Use Observer-Backed Read Models for Dashboard, Inbox, and Aggregate Task Views](adrs/adr-004.md) — Requires reusable dashboard and inbox hooks in the tasks system

## Deliverables
- `web/src/systems/tasks/` scaffold with types, adapter, query layer, hooks, and public barrel
- Route-hook orchestration for tasks page, detail, and run detail
- Adapter, query, and hook tests with >=80% coverage for the new scaffold **(REQUIRED)**
- No direct route/component calls to `apiClient` outside the tasks adapter layer **(REQUIRED)**
- A reusable tasks system that later UI tasks can consume without rebuilding data access

## Tests
- Unit tests:
  - [ ] Tasks adapter methods call the expected generated endpoints and normalize transport or contract errors consistently
  - [ ] Query keys and query options remain stable for list, detail, timeline, tree, run-detail, dashboard, and inbox reads
  - [ ] Query options propagate filters, identifiers, and live-read parameters into the generated client without manual shape conversion
  - [ ] Route hooks derive URL state, selected IDs, active mode, and panel state correctly from search params and route params
  - [ ] Mutation hooks invalidate the expected task queries after create, publish, approval, archive, dismiss, and mark-read actions
  - [ ] Shared formatters or type helpers normalize status, lane, and badge data without leaking raw transport enums into components
- Integration tests:
  - [ ] Route hooks and shared task hooks compose correctly with mocked data across base, detail, and run-detail routes
  - [ ] Query invalidation refreshes dependent task surfaces coherently after create, publish, and inbox-action mutations
  - [ ] Generated OpenAPI types align with the tasks adapter signatures without manual type escapes or `any` fallbacks
  - [ ] The tasks system public barrel exports remain sufficient for route consumers without deep-import drift
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80% for the new tasks system scaffold
- The web app has one coherent tasks domain layer for data access and orchestration
- Screen implementation tasks can focus on UI behavior instead of rebuilding task fetching logic
