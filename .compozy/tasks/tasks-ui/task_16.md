---
status: pending
title: Dashboard and inbox routes
type: frontend
complexity: high
dependencies:
  - task_13
---

# Task 16: Dashboard and inbox routes

## Overview

Implement the aggregate operator views for Tasks: dashboard and inbox. This task should turn the observer-backed dashboard and inbox reads into usable UI surfaces with cards, lanes, triage actions, filters, and navigation that match the Paper screens.

<critical>
- ALWAYS READ `_techspec.md`, ADRs, `task_13.md`, and the dashboard/inbox analysis docs before building aggregate views
- REFERENCE TECHSPEC sections "System Architecture", "Testing Approach", and "Monitoring and Observability"
- FOCUS ON "WHAT" — implement aggregate operator views, not the base task-detail/live routes
- MINIMIZE CODE — reuse dashboard and inbox hooks from the tasks system instead of route-local aggregation
- TESTS REQUIRED — aggregate cards, lane grouping, triage actions, and empty/error states all need coverage
- GREENFIELD: dashboard e inbox devem refletir read models proprios; nao recair em agrupamento manual no cliente para fechar a UI
</critical>

<requirements>
- MUST implement the dashboard view using the observer-backed task dashboard read model
- MUST implement the inbox view with lane grouping, unread/archive state, and triage/approval actions
- MUST keep aggregate fetch state and action orchestration inside the shared tasks hooks and route hook
- MUST provide loading, error, and empty states appropriate for operator dashboards and inboxes
- SHOULD support the filter/search state required by the Paper inbox and dashboard controls
</requirements>

## Subtasks
- [ ] 16.1 Extend the tasks page orchestration for dashboard and inbox modes
- [ ] 16.2 Implement dashboard cards, summaries, queue/health sections, and recent activity panels
- [ ] 16.3 Implement inbox lanes, task cards, and approval/triage actions
- [ ] 16.4 Add route and component tests for aggregate loading, action, and empty-state behavior

## Implementation Details

See TechSpec sections "Monitoring and Observability", "Testing Approach", and the analysis docs for dashboard and inbox. These views should be built directly on the aggregate read models from the tasks system, not derived from the main task list.

### Relevant Files
- `web/src/routes/_app/tasks.tsx` — base tasks area route that will host dashboard/inbox modes
- `web/src/hooks/routes/use-tasks-page.ts` — page orchestration for aggregate modes, filters, and actions
- `web/src/systems/tasks/hooks/use-task-dashboard.ts` — dashboard-specific reads introduced by the tasks system
- `web/src/systems/tasks/hooks/use-task-inbox.ts` — inbox-specific reads and actions introduced by the tasks system
- `web/src/systems/tasks/components/` — new dashboard and inbox components introduced by this task
- `.compozy/tasks/tasks-ui/analysis/analysis_dashboard.md` — dashboard gap analysis and operator-read requirements
- `.compozy/tasks/tasks-ui/analysis/analysis_inbox.md` — inbox lanes, approval, and triage requirements
- `docs/design/paper/tasks/` — local Paper exports for dashboard and inbox

### Dependent Files
- `web/src/routes/_app/-tasks.test.tsx` — route-level dashboard/inbox coverage
- `web/src/systems/tasks/**/*.test.tsx` — component and hook coverage for aggregate views
- `web/e2e/tasks.spec.ts` or `web/e2e/tasks-*.spec.ts` — browser QA in task_19 will exercise dashboard/inbox navigation and actions
- `web/src/systems/workspace/components/workspace-page-shell.tsx` — aggregate layout should remain aligned with the shared operator shell

### Related ADRs
- [ADR-004: Use Observer-Backed Read Models for Dashboard, Inbox, and Aggregate Task Views](adrs/adr-004.md) — Dashboard and inbox must consume dedicated aggregate read models

## Deliverables
- Dashboard and inbox operator views inside the Tasks area
- Approval and triage action handling wired through the shared tasks system
- Route and component tests with >=80% coverage for aggregate view behavior **(REQUIRED)**
- Loading, error, and empty states for dashboard and inbox surfaces **(REQUIRED)**
- No client-side reconstruction of dashboard or inbox aggregates from raw list data

## Tests
- Unit tests:
  - [ ] Dashboard cards render queue depth, health, totals, recent activity, and warning states from the aggregate read model
  - [ ] Inbox lanes render the expected grouping, counts, ordering, and action affordances for approvals, blocked items, failed runs, and archived work
  - [ ] Aggregate mode and lane filters map search params into the expected dashboard or inbox queries
  - [ ] Approval and triage actions invalidate and refresh inbox data correctly without stale badge or count state
  - [ ] Aggregate loading, error, and empty states render without layout instability or list-view assumptions leaking in
- Integration tests:
  - [ ] The tasks route can switch into dashboard and inbox modes without leaking list or kanban assumptions into aggregate layouts
  - [ ] Inbox actions update lane state, unread counts, and visible item placement coherently after approval, archive, dismiss, or mark-read flows
  - [ ] Dashboard and inbox hooks integrate correctly with the route orchestrator and generated task types
  - [ ] Aggregate mode selection and filter state survive navigation or refresh without resetting to list defaults unexpectedly
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80% for modified aggregate-view files
- Operators can use dashboard and inbox as first-class aggregate views inside Tasks
- The UI consumes dedicated aggregate read models instead of rebuilding dashboard/inbox semantics locally
