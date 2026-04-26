---
status: pending
title: Tasks UI Manual First Labels And E2E
type: frontend
complexity: medium
dependencies:
  - task_10
  - task_14
---

# Task 15: Tasks UI Manual First Labels And E2E

## Overview
Update the existing Tasks UI so users understand the manual-first execution boundary and coordinator handoff. This is an honesty pass over current task surfaces, not a broad autonomy dashboard.

<critical>
- ALWAYS READ `_techspec.md`, ADR-010, ADR-011, `DESIGN.md`, `web/AGENTS.md`, and current task system files before changing UI
- DO NOT BUILD A NEW AUTONOMY DASHBOARD IN MVP
- UI COPY MUST MAKE CREATION VS START/PUBLISH/APPROVAL CLEAR
- USE GENERATED CONTRACT TYPES - do not hand-roll duplicate frontend DTOs
- TESTS REQUIRED - adapter, component, and Playwright coverage for task lifecycle labels/actions
- NO WORKAROUNDS - do not hide backend ambiguity with vague UI copy; fix contracts or labels at the source
</critical>

<requirements>
- MUST update existing task list/detail/action UI to distinguish draft/created tasks from executable queued/running work.
- MUST label publish/start/approval actions as the coordinator handoff boundary when coordinator is enabled.
- MUST preserve manual operator control: users can create tasks, start them, and start sessions independently.
- MUST consume generated contract fields from task_10/task_14 instead of local type guesses.
- MUST add or update Playwright E2E coverage for manual-first task creation/start and coordinator-trigger labels.
- MUST not add scheduler dashboards, coordinator admin systems, config UI, or marketing pages in this MVP task.
</requirements>

## Subtasks
- [ ] 15.1 Audit `web/src/systems/tasks` for task lifecycle labels, action affordances, and generated type usage.
- [ ] 15.2 Update adapters/hooks/components to represent created, queued, running, completed, failed, and approval-needed states accurately.
- [ ] 15.3 Update action labels/tooltips so start/publish/approval clearly means executable run/coordinator handoff.
- [ ] 15.4 Add mocks/fixtures for user-created task, agent-created pending approval, queued run, and coordinator-enabled workspace.
- [ ] 15.5 Add Playwright E2E for create-without-run, start-enqueues-run, approval-enqueues-run, and manual session coexistence labels.
- [ ] 15.6 Run web lint, typecheck, unit tests, and E2E gates required by `web/AGENTS.md`.

## Implementation Details
Stay inside the existing Tasks system and design tokens. The goal is to prevent operator confusion: creating a task should read as saved intent; start/publish/approve should read as making it executable and eligible for coordinator orchestration.

Do not create a card-heavy autonomy overview or a scheduler/coordinator monitoring surface. Those are post-MVP web tasks.

### Relevant Files
- `DESIGN.md` - authoritative design tokens and UI constraints.
- `web/AGENTS.md` - frontend-specific project rules and verification gates.
- `web/src/systems/tasks/types.ts` - task/run frontend types and generated contract usage.
- `web/src/systems/tasks/adapters/tasks-api.ts` - API-to-UI mapping.
- `web/src/systems/tasks/hooks/use-task-actions.ts` - lifecycle action hooks.
- `web/src/systems/tasks/components/tasks-detail-header.tsx` - visible task status/actions.
- `web/src/systems/tasks/components/*` - task list/detail components affected by labels.
- `web/src/systems/tasks/mocks/*` - fixtures for UI and E2E state.
- `web/e2e/tasks.spec.ts` - Playwright coverage for task flows.
- `.resources/multica/e2e/issues.spec.ts` - reference for task/issue E2E coverage style.
- `.resources/paperclip/doc/plans/2026-04-07-issue-detail-speed-and-optimistic-inventory.md` - reference for issue detail UI and optimistic state risks.

### Dependent Files
- `packages/site/content/runtime/core/autonomy/` - task_16 documents UI-visible behavior.
- `.compozy/tasks/autonomous/qa/test-cases/` - task_17 plans UI QA cases.

### Related ADRs
- [ADR-010: Manual Operator Control Remains First-Class](adrs/adr-010.md) - UI must preserve manual workflows.
- [ADR-011: Generated Contract And Runtime Docs Co-Ship](adrs/adr-011.md) - generated frontend contract usage.

## Deliverables
- Updated Tasks UI labels/actions for creation versus execution start.
- Updated adapters/types/mocks aligned with generated backend contracts.
- Playwright E2E coverage for manual-first and coordinator-trigger task flows.
- Frontend unit tests with 80%+ coverage for touched adapters/hooks/components **(REQUIRED)**.
- Passing web lint/typecheck/test/E2E gates for changed surfaces **(REQUIRED)**.

## Tests
- Unit tests:
  - [ ] Task adapter maps created/draft tasks without runs to non-executable saved-intent labels.
  - [ ] Task adapter maps queued/running runs to executable/coordinator-handoff labels.
  - [ ] Action hook chooses create, start, publish, approve, or retry actions based on generated state fields.
  - [ ] Component tests assert no label implies that creation alone starts autonomy.
  - [ ] Mocks cover user-created, agent-created approval, queued, running, failed, and coordinator-enabled states.
- Integration/E2E tests:
  - [ ] Creating a task in the UI does not show a queued/running run until start/publish.
  - [ ] Starting a task shows queued/coordinator-handoff state and preserves manual control copy.
  - [ ] Approving an agent-created task transitions into executable queued state.
  - [ ] Existing manual session start UI is unaffected by task autonomy labels.
  - [ ] `make web-lint`, `make web-typecheck`, `make web-test`, and relevant Playwright task specs pass.
- Test coverage target: >=80% for changed frontend files.
- All tests must pass.

## Success Criteria
- All tests passing.
- Test coverage >=80% for changed frontend files.
- UI clearly communicates that task creation and autonomous execution are separate steps.
- No broad autonomy dashboard or post-MVP web scope is introduced.
