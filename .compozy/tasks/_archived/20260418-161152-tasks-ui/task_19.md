---
status: completed
title: Tasks QA execution and settings-aligned browser E2E
type: test
complexity: critical
dependencies:
  - task_18
---

# Task 19: Tasks QA execution and settings-aligned browser E2E

## Overview

Execute the full QA pass for Tasks using the artifacts from `task_18`, and commit durable browser E2E coverage that follows the repo’s daemon-served Playwright pattern. This task is also responsible for bringing Settings browser coverage up to the same standard on the execution branch, so Tasks and Settings both participate properly in the shared `web/e2e` lane.

<critical>
- ALWAYS READ `_techspec.md`, ADRs, and the QA artifacts from `task_18` before running any validation
- ACTIVATE `/qa-execution` with `qa-output-path=.compozy/tasks/tasks-ui` before any live verification or evidence capture
- IF QA FINDS A BUG, ACTIVATE `/systematic-debugging` AND `/no-workarounds` BEFORE CHANGING CODE OR TESTS
- FOLLOW THE PROJECT E2E PATTERN — use the existing daemon-served Playwright harness under `web/e2e/`; do not replace it with one-off browser scripts or dev-server-only checks
- FOCUS ON SHIPPED OPERATOR FLOWS — Tasks browser coverage and Settings browser coverage both need durable proof inside the normal repo lane
- IF THE SETTINGS SURFACE IS NOT PRESENT ON THE EXECUTION BRANCH, TREAT THAT AS A BLOCKER AND REPORT IT EXPLICITLY; DO NOT FAKE OR SKIP THE COVERAGE SILENTLY
- DO NOT WEAKEN TESTS TO GET GREEN — fix production code or configuration at the source, then rerun the affected scenarios and full gates
- GREENFIELD: a cobertura E2E de tasks e settings precisa entrar no fluxo normal do projeto (`make test-e2e-web` / `make verify`), nao ficar em checks paralelos ou manuais
</critical>

<requirements>
- MUST use the `/qa-execution` skill with `qa-output-path=.compozy/tasks/tasks-ui`
- MUST consume `.compozy/tasks/tasks-ui/qa/test-plans/` and `.compozy/tasks/tasks-ui/qa/test-cases/` from `task_18` as the execution matrix seed
- MUST add committed daemon-served Playwright coverage for the Tasks feature under `web/e2e/`
- MUST add or extend committed daemon-served Playwright coverage for the Settings feature under `web/e2e/` so it follows the same project pattern
- MUST cover at least these persistent Tasks flows in browser E2E: open Tasks from the sidebar, create a draft task, publish it, inspect the task in split/detail flow, open run detail, and validate dashboard/inbox or live-state navigation
- MUST cover at least these persistent Settings flows in browser E2E when the surface exists on the execution branch: settings shell navigation, one restart-aware save flow, one collection CRUD flow, and one advanced settings flow such as workspace-scoped MCP or hooks/extensions
- MUST write fresh QA evidence to `.compozy/tasks/tasks-ui/qa/verification-report.md` and capture bugs/screenshots under the same artifact root
- MUST rerun the repository verification gates after the last fix, including `make test-e2e-web` and `make verify`
- SHOULD reuse `web/e2e/fixtures/test.ts`, `runtime.ts`, `runtime-seed.ts`, and `selectors.ts` rather than inventing a separate Tasks or Settings harness
</requirements>

## Subtasks
- [x] 19.1 Activate `/qa-execution` with `qa-output-path=.compozy/tasks/tasks-ui` and derive the execution matrix from `task_18` artifacts
- [x] 19.2 Extend shared Playwright selector/runtime-seed helpers for Tasks and Settings only where the existing `web/e2e` pattern needs explicit support
- [x] 19.3 Implement daemon-served browser E2E specs for the critical Tasks and Settings operator flows, or report the explicit Settings blocker when the shipped surface is absent on the branch
- [x] 19.4 Execute CLI, API, and browser QA flows, capture evidence/bugs, and fix root-cause regressions with matching regression tests
- [x] 19.5 Rerun `make test-e2e-web`, `make verify`, and publish `.compozy/tasks/tasks-ui/qa/verification-report.md`

## Implementation Details

See TechSpec sections "Testing Approach", "Verification gates", "Development Sequencing", and "Known Risks". The key constraint is that Tasks QA and Settings regression coverage must both become part of the repo’s standard browser lane instead of one-off exploratory checks.

### Relevant Files
- `.agents/skills/qa-execution/SKILL.md` — required workflow for execution matrix discovery, evidence capture, and verification reporting
- `.compozy/tasks/tasks-ui/qa/test-plans/` — planning artifacts that seed the QA execution matrix
- `.compozy/tasks/tasks-ui/qa/test-cases/` — manual cases and priorities that the execution pass must honor
- `scripts/discover-project-contract.py` — canonical repo-contract discovery entrypoint required by `/qa-execution`
- `Makefile` — repository-defined `test-e2e-web` and `verify` entrypoints that must pass before completion
- `web/playwright.config.ts` — shared Playwright configuration for the daemon-served browser lane
- `web/e2e/fixtures/test.ts` — canonical browser fixture entrypoint used by repo E2E specs
- `web/e2e/fixtures/runtime.ts` — daemon-served runtime harness for browser E2E
- `web/e2e/fixtures/runtime-seed.ts` — seeded runtime helpers that new Tasks/Settings flows should extend instead of duplicating
- `web/e2e/fixtures/selectors.ts` — shared selector helpers that Tasks or Settings coverage should expand if needed
- `web/src/routes/_app/tasks*.tsx` — shipped Tasks route surfaces that the new E2E specs must exercise
- `web/src/routes/_app/settings*.tsx` — shipped Settings route surfaces that the regression E2E must exercise when present on the branch

### Dependent Files
- `web/e2e/tasks.spec.ts` or `web/e2e/tasks-*.spec.ts` — committed daemon-served Tasks browser E2E coverage
- `web/e2e/settings.spec.ts` or `web/e2e/settings-*.spec.ts` — committed daemon-served Settings browser E2E coverage
- `web/e2e/fixtures/selectors.ts` — may gain stable Tasks and Settings selectors for the new flows
- `web/e2e/fixtures/runtime-seed.ts` — may gain deterministic seed helpers for Tasks drafts/runs and Settings prerequisites
- `.compozy/tasks/tasks-ui/qa/verification-report.md` — final QA evidence written by `/qa-execution`
- `.compozy/tasks/tasks-ui/qa/screenshots/` — browser evidence for the executed Tasks and Settings flows
- `.compozy/tasks/tasks-ui/qa/issues/BUG-*.md` — structured bug reports for failures discovered during execution

### Related ADRs
- [ADR-001: First-Class Tasks Area in the Main App Shell](adrs/adr-001.md) — Browser E2E must validate Tasks as a primary operator surface
- [ADR-003: Add Dedicated Task Live Surfaces Instead of Client-Side Stitching](adrs/adr-003.md) — Tasks E2E must validate the task-native live/detail experience
- [ADR-004: Use Observer-Backed Read Models for Dashboard, Inbox, and Aggregate Task Views](adrs/adr-004.md) — Tasks E2E must validate aggregate dashboard/inbox behavior

## Deliverables
- Fresh `.compozy/tasks/tasks-ui/qa/verification-report.md` produced by `/qa-execution`
- Committed daemon-served Playwright Tasks E2E coverage under `web/e2e/` **(REQUIRED)**
- Committed daemon-served Playwright Settings E2E coverage or an explicit blocker report if the Settings surface is not yet present on the execution branch **(REQUIRED)**
- Shared browser fixture/selector support only where needed by the new E2E specs **(REQUIRED)**
- Root-cause bug fixes plus matching regression tests for any issues discovered during execution **(REQUIRED)**
- Fresh screenshots and bug reports under `.compozy/tasks/tasks-ui/qa/` for the executed flows **(REQUIRED)**
- Passing `make test-e2e-web` and `make verify` after the final QA fixes **(REQUIRED)**

## Tests
- Unit tests:
  - [x] New Tasks and Settings browser selector helpers resolve stable shell, panel, table, and form surfaces without brittle text-only targeting
  - [x] Runtime seed helpers can create deterministic prerequisites for draft tasks, runnable tasks, and run-detail scenarios
  - [x] Runtime seed helpers also create deterministic Settings prerequisites when the Settings surface exists on the execution branch, or the branch blocker is documented explicitly when it does not
  - [x] Shared fixture or evidence helpers write screenshots and report artifacts into the expected QA output paths
- Integration tests:
  - [x] Tasks Playwright coverage exercises sidebar entry, draft creation, publication, detail inspection, and run-detail navigation
  - [x] Tasks Playwright coverage exercises dashboard and inbox navigation, or explicitly validated live-view fallback behavior, using the daemon-served harness
  - [x] Tasks browser coverage captures screenshots or evidence for the critical flows under `.compozy/tasks/tasks-ui/qa/`
  - [x] Settings Playwright coverage exercises the required critical flows when the Settings surface exists on the execution branch, or the missing surface is reported as an explicit blocker when it does not
  - [x] `make test-e2e-web` passes with the new Tasks and Settings scenarios included in the browser lane
  - [x] `make verify` passes after the final QA fix set
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- The `/qa-execution` workflow has been run explicitly with artifacts stored under `.compozy/tasks/tasks-ui/qa/`
- The Tasks feature has committed browser E2E coverage that follows the repo’s daemon-served Playwright pattern
- The Settings feature has committed browser E2E coverage in the same lane, or an explicit blocker has been documented if the surface is not yet present on the execution branch
- Any QA failures were fixed at the source and documented with fresh evidence
- The normal repo verification gates, including the browser E2E lane, pass with the new coverage in place

## Completion Notes

- Tasks browser coverage landed in `web/e2e/tasks.spec.ts` and participates in the shared daemon-served browser lane.
- Settings browser coverage remains blocked on this execution branch because `web/src/routes/_app/settings*.tsx` is absent; see `.compozy/tasks/tasks-ui/qa/issues/BUG-002-settings-surface-missing-on-branch.md`.
- Final QA evidence is published in `.compozy/tasks/tasks-ui/qa/verification-report.md`.
