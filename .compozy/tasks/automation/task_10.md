---
status: pending
title: Build web automation management UI
type: frontend
complexity: high
dependencies:
  - task_07
---

# Task 10: Build web automation management UI

## Overview

Build the `/automation` experience in the web app so users can inspect, create, edit, and monitor jobs and triggers from the existing AGH SPA. This task should follow the project's system-layer web conventions and consume the generated API types from the automation transport surface rather than inventing parallel client models.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
1. MUST add a web automation system with adapters, query infrastructure, hooks, and components consistent with the existing `web/src/systems/*` organization.
2. MUST add an `/automation` route and sidebar navigation entry so the feature is reachable from the current application shell.
3. MUST support list, detail, create/edit, manual trigger, run history, and scope-aware filtering flows for jobs and triggers.
4. SHOULD reuse generated OpenAPI types and shared API-client patterns instead of handwritten fetch contracts where the generated spec already provides coverage.
</requirements>

## Subtasks
- [ ] 10.1 Create the `web/src/systems/automation/` module with types, adapters, query keys, query options, and hooks
- [ ] 10.2 Add the `/automation` route and page shell composition
- [ ] 10.3 Add list, detail, and form components for jobs, triggers, and runs
- [ ] 10.4 Add sidebar navigation and route-active state for Automation
- [ ] 10.5 Add component, hook, and route tests for the main user flows

## Implementation Details

Follow the TechSpec sections "Web UI", "Impact Analysis", and the project's existing `web/src/systems/*` pattern used by skills, sessions, knowledge, and workspace. Keep API access in adapters, query state in hooks/lib, and route composition in the TanStack Router file under `web/src/routes/_app/`.

### Relevant Files
- `web/src/routes/_app/skills.tsx` — Good reference for a routed workspace-aware feature page composed from a system module
- `web/src/systems/skill/adapters/skill-api.ts` — Shows the expected API-client adapter shape using generated OpenAPI routes
- `web/src/systems/skill/index.ts` — Barrel export pattern for a system module
- `web/src/components/app-sidebar.tsx` — Navigation entries and active-route styling belong here
- `web/src/generated/agh-openapi.d.ts` — Generated API types should be reused by the new automation adapters

### Dependent Files
- `web/src/routeTree.gen.ts` — Route generation will update once the new automation route is added
- `internal/api/spec/spec.go` — The UI depends on the automation endpoints being present in generated API types from task 07

### Related ADRs
- [ADR-002: Unified Automation Model — Schedules and Triggers](adrs/adr-002.md) — The UI should present jobs and triggers as one automation feature surface

## Deliverables
- New `web/src/systems/automation/` module with adapters, hooks, and components
- `/automation` route and sidebar entry in the application shell
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration tests for main automation UI flows **(REQUIRED)**

## Tests
- Unit tests:
  - [ ] Automation adapters call the expected `/api/automation/*` endpoints and map the returned payload shapes correctly
  - [ ] Query keys and invalidation behavior distinguish jobs, triggers, runs, and workspace-scoped filters
  - [ ] Automation page components render loading, empty, and error states for jobs and triggers
- Integration tests:
  - [ ] Visiting `/automation` renders list and detail panes backed by mocked automation API responses
  - [ ] Sidebar navigation shows an Automation item and marks it active on the automation route
  - [ ] Creating or editing a workspace-scoped automation entry submits the expected scope and workspace payload fields
  - [ ] Manual trigger and run-history interactions update the UI after a successful mutation
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- The AGH SPA exposes a navigable automation management surface
- Web automation state is organized using the existing system/adapters/hooks/query conventions rather than ad hoc route-local fetching
