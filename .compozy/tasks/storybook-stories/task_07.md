---
status: completed
title: Stories for automation and bridges systems
type: frontend
complexity: high
dependencies:
  - task_05
---

# Task 7: Stories for automation and bridges systems

## Overview
Cover the two heaviest form-driven systems: automation (7 components) and bridges (7 components) — 14 total. These systems host multi-step forms, list panels, and modal dialogs that drive most of the app's admin flows; their stories must exercise success, loading, and error handler overrides to document both the happy path and validation failures.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST add stories under `web/src/systems/automation/components/stories/*` for: `automation-detail-panel`, `automation-editor-dialog`, `automation-form-primitives`, `automation-job-form`, `automation-list-panel`, `automation-run-history`, `automation-trigger-form`.
- MUST add stories under `web/src/systems/bridges/components/stories/*` for: `bridge-create-dialog`, `bridge-detail-panel`, `bridge-edit-dialog`, `bridge-empty-state`, `bridge-list-panel`, `bridge-provider-card`, `bridge-test-delivery-dialog`.
- MUST title stories `systems/automation/<Name>` and `systems/bridges/<Name>`.
- MUST keep each component to 2–5 stories and prioritize: `Default`, one error variant (server-side rejection via handler override), one loading or empty variant where applicable.
- MUST NOT duplicate form primitive markup — compose from `@agh/ui` and `web/src/components/ui/*` primitives (`Field`, `InputGroup`, `Button`).
- MUST override MSW handlers only for the single story that needs the override (loading/error), leaving `Default` on the global success set.
- MUST NOT mutate shared Zustand stores from inside stories; all state must be component-local for the story iframe.
</requirements>

## Subtasks
- [x] 7.1 Automation list + detail stories: `automation-list-panel` (Default + Empty + Error), `automation-detail-panel` (Default + Error).
- [x] 7.2 Automation form stories: `automation-editor-dialog`, `automation-job-form`, `automation-trigger-form`, `automation-form-primitives` (Default + Validation error per form).
- [x] 7.3 Automation run history story: `automation-run-history` (Default + Empty).
- [x] 7.4 Bridges list + empty stories: `bridge-list-panel` (Default), `bridge-empty-state` (Default), `bridge-provider-card` (Default + Disabled).
- [x] 7.5 Bridges dialog stories: `bridge-create-dialog`, `bridge-edit-dialog`, `bridge-test-delivery-dialog` (Default + Error).
- [x] 7.6 Bridges detail story: `bridge-detail-panel` (Default + Error).

## Implementation Details
Follow TechSpec "Core Interfaces" story template; reuse fixtures from `systems/automation/mocks` and `systems/bridges/mocks`. For error overrides on mutation endpoints (POST, PUT, DELETE), return `HttpResponse.json({ error: "validation_failed", details: [...] }, { status: 422 })` so the form can display realistic inline feedback.

### Relevant Files
- `web/src/systems/automation/components/*.tsx` — automation subjects.
- `web/src/systems/bridges/components/*.tsx` — bridges subjects.
- `web/src/systems/automation/mocks/`, `web/src/systems/bridges/mocks/` — fixture/handler source.
- `packages/ui/src/index.ts` and `web/src/components/ui/*` — primitives for composition.

### Dependent Files
- `web/src/systems/{automation,bridges}/components/stories/*` — 14 new files.
- `task_11` — verifies these stories build and render.

### Related ADRs
- [ADR-002: MSW + Shared Decorators for System Stories](adrs/adr-002.md) — Handler-override pattern.
- [ADR-004: Per-System Mocks Directory](adrs/adr-004.md) — Fixture source.

## Deliverables
- 14 new story files across automation and bridges.
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration tests for form + dialog rendering under handler overrides **(REQUIRED)**

## Tests
- Unit tests:
  - [ ] `automation-job-form` `Validation error` story surfaces the API-returned `validation_failed` message on submit.
  - [ ] `automation-list-panel` `Empty` story overrides GET to return `[]` and renders the empty-state affordance.
  - [ ] `bridge-create-dialog` `Error` story keeps the dialog open after a 422 response.
  - [ ] `bridge-provider-card` `Disabled` story renders the provider card with an `aria-disabled="true"` action.
- Integration tests:
  - [ ] All 14 stories index in `build-storybook` and render without warnings.
  - [ ] `automation-trigger-form` submit-path story fires exactly one handler invocation and the MSW log matches the expected path.
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- 14 story files present across both systems.
- No Zustand store leakage between stories.
- Every error story asserts the error UI renders (not just the request).
