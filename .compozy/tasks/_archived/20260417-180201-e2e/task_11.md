---
status: completed
title: Browser automation operator flow
type: test
complexity: high
dependencies:
  - task_08
  - task_04
---

# Task 11: Browser automation operator flow

## Overview

Add the browser E2E scenario that proves an operator can manage automation jobs and triggers through the shipped Automation UI and observe the downstream run effects. This task stays browser-visible while depending on the runtime automation/task lane from `task_04` for real execution truth.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
1. MUST add a Playwright scenario that covers creating or editing an automation job or trigger through the UI and observing resulting run history.
2. MUST trigger a real runtime execution path, using a manual trigger in the UI when available and an out-of-browser ingress helper only when the trigger flow requires it.
3. MUST assert browser-visible outcomes such as detail-panel state, run history, linked session visibility, and resulting transcript-oriented surfaces when they are part of the operator flow.
4. MUST keep runtime orchestration truth in `task_04`; this task must not recreate automation/task semantics inside the browser layer.
5. SHOULD keep job and trigger operator work in one task because the shipped route presents them as one coherent management surface.
</requirements>

## Subtasks
- [x] 11.1 Seed the browser/runtime harness with automation jobs, triggers, and visible run history prerequisites.
- [x] 11.2 Implement the browser scenario for job or trigger create/edit flows.
- [x] 11.3 Implement real execution stimulation for the selected automation flow and observe run history in the UI.
- [x] 11.4 Add assertions for linked session or transcript surfaces when the operator flow exposes them.
- [x] 11.5 Add minimal selector or detail-panel stabilization only where the shipped route needs it.

## Implementation Details

See TechSpec sections "PR-Required Browser E2E", "Daemon-Only E2E In Current Product", and "Combined-Flow Follow-Up". This task proves operator workflows on the Automation page without replacing the runtime automation/task proofs.

### Relevant Files
- `web/src/routes/_app/automation.tsx` — primary browser route for automation jobs, triggers, and run history.
- `web/src/systems/automation/components/automation-detail-panel.tsx` — detail-panel surface that should reflect run and selection state.
- `web/src/systems/automation/components/automation-editor-dialog.tsx` — create/edit UI surface for jobs and triggers.
- `web/src/systems/automation/hooks/use-automation-actions.ts` — browser-side mutation paths for create, update, and manual trigger flows.
- `web/src/systems/automation/adapters/automation-api.ts` — transport surface used by the route.
- `web/e2e/fixtures/runtime.ts` — shared Playwright/browser fixture that seeds automation-enabled runtime state.

### Dependent Files
- `web/e2e/automation.spec.ts` — new browser automation operator scenario.
- `web/e2e/fixtures/selectors.ts` — optional shared selectors if route-level stability helpers are required.
- `web/src/routes/_app/automation.tsx` — may need minimal test-surface stabilization if the current UI handles are insufficient.
- `internal/daemon/daemon_integration_test.go` — runtime automation/task truth source that later browser cross-checks may reference sparingly.
- `Makefile` — later browser lane wiring must include this scenario in the browser E2E target set.

### Related ADRs
- [ADR-002: Separate Runtime and Browser E2E Lanes](adrs/adr-002.md) — This task is a browser-lane operator flow over runtime-managed automation state.
- [ADR-004: Assert Through Domain-Specific Product Surfaces](adrs/adr-004.md) — Browser-visible run history and linked session surfaces are the primary assertions here.
- [ADR-005: Keep PR-Required E2E On Shipped Surfaces and Use Tiered Execution](adrs/adr-005.md) — Automation management is an in-scope shipped browser surface.

## Deliverables
- Browser E2E scenario for automation job/trigger management and visible run history
- Browser/runtime fixture support for real automation execution in a Playwright run
- Minimal UI stabilization required for reliable browser assertions on the Automation route
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration tests for the browser automation operator workflow **(REQUIRED)**

## Tests
- Unit tests:
  - [x] Automation browser fixture seeding provides deterministic jobs, triggers, and run-history state
  - [x] Selector helpers locate create/edit surfaces, detail panels, and run-history panels consistently
  - [x] Browser artifact capture preserves enough route and run context to explain failed automation flows
- Integration tests:
  - [x] Operator can create or edit an automation job or trigger and see the updated state reflected in the UI
  - [x] Operator can cause a real automation execution and observe the resulting run history in the Automation page
  - [x] Operator can inspect linked session or transcript surfaces when the executed automation flow exposes them
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- Browser E2E proves an operator can manage and observe automation through the shipped UI
- Real execution is driven through the runtime lane rather than mocked in-page behavior
- The browser automation task remains decoupled from daemon-truth orchestration logic
