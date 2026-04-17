---
status: pending
title: Tasks QA plan and regression artifacts
type: docs
complexity: high
dependencies:
  - task_11
  - task_14
  - task_15
  - task_16
  - task_17
---

# Task 18: Tasks QA plan and regression artifacts

## Overview

Generate the reusable QA planning artifacts for the full Tasks feature before live execution begins, while explicitly pulling Settings into the regression matrix as a critical adjacent operator surface. This task should leave the repo with concrete test plans, route-by-route manual cases, and regression-suite definitions that the follow-up execution task can consume without redefining scope or output paths.

<critical>
- ALWAYS READ `_techspec.md`, ADRs, and `task_14.md` through `task_17.md` before planning coverage
- ACTIVATE `/qa-report` with `qa-output-path=.compozy/tasks/tasks-ui` before writing or revising any QA artifact
- KEEP THE SAME `qa-output-path` FOR `/qa-execution` — all planning and execution artifacts must live under `.compozy/tasks/tasks-ui/qa/`
- FOCUS ON "WHAT" — define coverage, risks, and evidence layout; do not execute product flows or pre-emptively fix bugs in this task
- USE THE LOCAL DESIGN EXPORTS — derive UI coverage from `docs/design/paper/tasks/` and `docs/design/paper/settings/` when Figma is unavailable
- GREENFIELD: tasks e settings precisam de rastreabilidade explicita na matriz de regressao; nao aceite smoke generico que deixa rotas/operator flows sem dono
</critical>

<requirements>
- MUST use the `/qa-report` skill with `qa-output-path=.compozy/tasks/tasks-ui`
- MUST generate a feature-level test plan under `.compozy/tasks/tasks-ui/qa/test-plans/`
- MUST generate manual test cases covering dashboard, inbox, list, kanban, empty state, create modal, detail timeline, run detail, and multi-agent live
- MUST include Settings as regression-critical coverage in the plan, especially for browser E2E expectations that follow the repo’s `web/e2e` pattern
- MUST produce at least one regression-suite document defining smoke, targeted, and full execution priorities for Tasks plus Settings-adjacent regression coverage
- SHOULD derive UI-focused cases from the local Paper exports when Figma MCP is unavailable
</requirements>

## Subtasks
- [ ] 18.1 Activate `/qa-report` with `qa-output-path=.compozy/tasks/tasks-ui`
- [ ] 18.2 Write the feature-level Tasks test plan with scope, risks, environments, and entry/exit criteria
- [ ] 18.3 Generate route-by-route manual test cases for Tasks and the required Settings regression-critical flows
- [ ] 18.4 Build regression-suite definitions and identify the P0/P1 flows that `/qa-execution` must run first
- [ ] 18.5 Validate artifact completeness, traceability, and handoff readiness for `task_19`

## Implementation Details

See TechSpec sections "Testing Approach", "Known Risks", and "Development Sequencing". This task is the formal handoff from implementation to QA execution: it should capture what must be proven for Tasks, which Settings flows are mandatory regression coverage, and where all evidence must live.

### Relevant Files
- `.agents/skills/qa-report/SKILL.md` — required planning workflow, output structure, and artifact naming rules
- `.compozy/tasks/tasks-ui/_techspec.md` — source of truth for Tasks routes, live behavior, and verification expectations
- `.compozy/tasks/tasks-ui/task_14.md` — defines the main list/kanban/empty/create workflows that QA must trace
- `.compozy/tasks/tasks-ui/task_15.md` — defines task detail and run-detail deep-link behavior that QA must trace
- `.compozy/tasks/tasks-ui/task_16.md` — defines dashboard and inbox aggregate workflows that QA must trace
- `.compozy/tasks/tasks-ui/task_17.md` — defines the multi-agent live surface that QA must trace
- `docs/design/paper/tasks/` — local Paper exports for the Tasks screens
- `docs/design/paper/settings/` — local Paper exports for the Settings regression-critical screens

### Dependent Files
- `.compozy/tasks/tasks-ui/qa/test-plans/tasks-ui-test-plan.md` — feature-level QA plan created by this task
- `.compozy/tasks/tasks-ui/qa/test-plans/*-regression.md` — regression-suite document(s) consumed by the execution task
- `.compozy/tasks/tasks-ui/qa/test-cases/TC-*.md` — manual test cases with priorities and expected results
- `.compozy/tasks/tasks-ui/qa/issues/BUG-*.md` — only created if planning uncovers a concrete documented discrepancy
- `.compozy/tasks/tasks-ui/task_19.md` — execution task that must consume this artifact set unchanged

### Related ADRs
- [ADR-001: First-Class Tasks Area in the Main App Shell](adrs/adr-001.md) — QA planning must treat Tasks as one primary operator surface
- [ADR-003: Add Dedicated Task Live Surfaces Instead of Client-Side Stitching](adrs/adr-003.md) — QA planning must distinguish task-native live flows from session drill-downs
- [ADR-004: Use Observer-Backed Read Models for Dashboard, Inbox, and Aggregate Task Views](adrs/adr-004.md) — QA planning must cover aggregate dashboard/inbox behavior explicitly

## Deliverables
- `.compozy/tasks/tasks-ui/qa/test-plans/tasks-ui-test-plan.md`
- One or more `.compozy/tasks/tasks-ui/qa/test-plans/*-regression.md` documents reusable by `/qa-execution`
- Route-by-route manual test cases under `.compozy/tasks/tasks-ui/qa/test-cases/` **(REQUIRED)**
- Explicit P0/P1 regression coverage for Tasks plus required Settings browser-regression flows **(REQUIRED)**
- A stable artifact layout under `.compozy/tasks/tasks-ui/qa/` that the execution task can consume without path changes **(REQUIRED)**

## Tests
- Unit tests:
  - [ ] `tasks-ui-test-plan.md` includes objectives, scope, environment matrix, entry/exit criteria, risk assessment, and explicit artifact ownership
  - [ ] Manual test cases exist for dashboard, inbox, list, kanban, empty state, create modal, detail timeline, run detail, and multi-agent live
  - [ ] Required Settings regression-critical browser flows are represented explicitly rather than being implied by a generic regression note
  - [ ] Each manual test case includes preconditions, steps, expected results, and priority or severity metadata suitable for execution
  - [ ] Regression-suite documents identify smoke, targeted, and full coverage plus execution order and blocker expectations for P0/P1 flows
- Integration tests:
  - [ ] All generated artifacts land under `.compozy/tasks/tasks-ui/qa/` and can be consumed directly by `/qa-execution`
  - [ ] Test cases trace back to the relevant Tasks routes, Paper screens, or regression-critical Settings flows clearly
  - [ ] Regression-suite and test-plan documents reference the same case IDs, priorities, and artifact paths without naming drift
  - [ ] Any bug report created during planning references the originating test case or design discrepancy clearly
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- The `/qa-report` workflow has been executed explicitly and its artifacts are stored under `.compozy/tasks/tasks-ui/qa/`
- Every Tasks screen and required Settings regression flow has at least one traceable QA artifact
- `task_19` can start execution without redefining scope, output paths, or risk priorities
- The Tasks feature has a concrete regression plan instead of ad hoc QA notes
