---
status: completed
title: "Web UI for Execution Profiles, Review State, and Notification Diagnostics"
type: frontend
complexity: critical
dependencies:
  - task_26
---

# Task 27: Web UI for Execution Profiles, Review State, and Notification Diagnostics

## Overview
This task implements the actual web UI for orchestration profiles, review queue/state, verdict actions, continuation guidance, and notification diagnostics. It must use AGH design tokens, truthful runtime state, and responsive layouts without inventing unsupported controls.

<critical>
- ALWAYS READ `_techspec.md`, `_techspec_orchestration.md`, `_techspec_review_gate.md`, every ADR, and the dependency task files before starting.
- REFERENCE TECHSPEC for implementation details; do not duplicate architecture or code snippets here.
- FOCUS ON WHAT needs to be delivered, keep changes scoped, and avoid compatibility shims or fallback paths.
- TESTS REQUIRED: every production change must ship focused tests and run the listed verification gates.
</critical>

<requirements>
- MUST add UI surfaces for execution profile inspection/edit/delete using task data hooks.
- MUST show review request/status/outcome/guidance and reviewer verdict states truthfully.
- MUST show notification subscription/cursor diagnostics and SSE resume state without fake metrics.
</requirements>

## Subtasks
- [x] Read `web/CLAUDE.md`, `DESIGN.md`, `COPY.md`, and `docs/_memory/glossary.md` before writing UI.
- [x] Implement profile management components and route integration.
- [x] Implement review queue/detail/verdict/continuation components with disabled/read-only states.
- [x] Implement notification diagnostics panel and stream resume status.
- [x] Add component tests, route tests, visual sanity checks, and Playwright coverage.

## Implementation Details
Required skill activation must match the touched surfaces: backend tasks use `agh-code-guidelines`, `golang-pro`, and `agh-test-conventions`; contract tasks also use `agh-contract-codegen-coship`; web tasks use the web instructions and frontend/design skills; docs tasks use `documentation-writer`, `copywriting` when public prose changes, and the site instructions; QA tasks use the QA skills named in the task. Use the TechSpec and ADRs for architecture; this task records scope and evidence boundaries.

### Relevant Files
- `web/src/systems/tasks/components` - task UI components.
- `web/src/systems/tasks/routes` or route integration files - task pages.
- `web/src/systems/tasks/hooks` - data hooks from task 26.
- `web/e2e` - Playwright coverage.

### Dependent Files
- `DESIGN.md` - visual tokens and UI grammar.
- `COPY.md` - labels and claim standards.
- `web/CLAUDE.md` - frontend rules.
- `web/src/generated/agh-openapi.d.ts` - runtime truth source.

### Related ADRs
- [ADR-003: Durable Cursor Primitive](adrs/adr-003.md) - notification cursor and replay semantics.
- [ADR-005: Denormalized Current Run Projection](adrs/adr-005.md) - current run projection boundaries.
- [ADR-007: Post-Terminal Review Gate](adrs/adr-007.md) - review request/verdict/continuation authority.
- [ADR-010: Typed Overlay](adrs/adr-010.md) - execution profile schema and config overlay shape.

## Extensibility / Agent Manageability / Config Lifecycle
- Extensibility: Makes orchestration/review/notification runtime state inspectable and manageable from the web shell.
- Agent manageability: Operators can manage profiles/reviews and inspect notifications without diverging from CLI/API authority.
- Config lifecycle: No config writes unless backed by existing profile API; avoid fake config controls.

### Web/Docs Impact
- `web/`: Primary task: `web/src/systems/tasks/**`, tests, and e2e.
- `packages/site`: Screenshots or UI behavior can be referenced by docs in tasks 28 and 29.

## Deliverables
- Task implementation or documentation matching the requirements above.
- Focused unit tests with 80%+ coverage where code changes.
- Integration, contract, e2e, or docs-build tests proportional to the touched behavior.
- Updated workflow memory, QA evidence, generated artifacts, or site docs when applicable.

## Tests
- Unit tests:
  - [ ] Validate the primary success path for this task.
  - [ ] Validate malformed input, missing dependency, or authorization failure paths.
  - [ ] Validate boundary conditions named by the related TechSpec and ADRs.
- Integration tests:
  - [ ] Exercise the task through the owning service/transport boundary when applicable.
  - [ ] Compare persisted state, generated contract output, or rendered docs/UI with runtime truth.
  - [ ] Run race, codegen, site, web, or full verify gates listed by the touched surface.
- Test coverage target: >=80% for changed code paths; docs-only tasks require 100% checklist evidence against authored pages.
- All tests must pass.

## Completion Evidence
- Web orchestration tab + run-detail reviews card shipped under `web/src/systems/tasks/components/tasks-{execution-profile,reviews,bridge-notifications,stream-resume,detail-orchestration}-card.tsx` plus the composite `TasksDetailOrchestrationPanel` and the `Orchestration` tab on `web/src/routes/_app/tasks.$id.tsx`.
- Run-level reviews appear under `web/src/routes/_app/tasks.$id.runs.$runId.tsx` via the shared `TasksReviewsCard` (`testId="tasks-run-reviews-card"`).
- Route data + mutations live in `web/src/hooks/routes/use-task-detail-orchestration-tab.ts` and `web/src/hooks/routes/use-task-detail-route.ts`; run-page data adds reviews via `useTaskRunReviews` with an `enableRunReviews` flag.
- Profile editor state extracted into `web/src/systems/tasks/hooks/use-profile-editor.ts` to satisfy the project hook/component lint.
- 2026-05-05 audit follow-up: `useTaskDetailOrchestrationTab` now resets `streamState`/`streamErrorMessage` via a `useEffect` keyed on `streamEnabled`, so opening the orchestration tab moves the UI from `"disabled"` to `"idle"` immediately while `useTaskStream`'s new subscription awaits its first frame; closing the tab snaps back to `"disabled"` and drops any prior error text. Regression coverage in `web/src/hooks/routes/use-task-detail-orchestration-tab.test.tsx` exercises both transitions. The Playwright spec was updated to assert the truthful seeded-profile summary (`tasks-execution-profile-summary`) rather than the empty state, since `seedBrowserTasksOperatorFlow` lands tasks with the runtime's default `inherit` profile.
- Tests: focused Vitest for the orchestration-tab hook (1 file / 6 tests), broader hook+systems suite (61 files / 427 tests), and the workspace Playwright command `bun run test:e2e:daemon-served:raw -- e2e/tasks-orchestration.spec.ts` PASS (1 spec, 1.3s) after building `.tmp/agh` + `.tmp/acpmock-driver` and exporting `AGH_TEST_DAEMON_BIN` / `AGH_TEST_ACPMOCK_DRIVER_BIN`. `make web-lint` / `make web-typecheck` / `make web-test` (212/1629) / `make web-build` clean. `make verify` PASS — Go race gate `DONE 8283 tests in 62.458s`, `OK: all package boundaries respected`.
- Workflow memory updated in `.compozy/tasks/orch-improvs/memory/task_27.md`.

## Success Criteria
- All tests passing.
- Test coverage >=80% for changed code paths, or 100% documented evidence coverage for docs-only tasks.
- `make verify` passes before the task is marked complete.
- The task evidence is recorded in workflow memory or QA artifacts.
