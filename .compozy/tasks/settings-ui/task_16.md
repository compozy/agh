---
status: pending
title: Settings QA execution and daemon-served browser E2E
type: test
complexity: critical
dependencies:
  - task_15
---

# Task 16: Settings QA execution and daemon-served browser E2E

## Overview

Execute the full QA pass for the settings feature using the planned artifacts from `task_15`, and commit durable browser E2E coverage that follows the repo's existing daemon-served Playwright pattern. This task is the quality gate for the entire settings surface: it must validate the shipped workflows like a real operator, fix root-cause regressions, and leave behind repeatable regression coverage in the normal `web/e2e` lane.

<critical>
- ALWAYS READ `_techspec.md`, ADRs, and the QA artifacts from `task_15` before running any validation
- ACTIVATE `/qa-execution` with `qa-output-path=.compozy/tasks/settings-ui` before any live verification or evidence capture
- IF QA FINDS A BUG, ACTIVATE `/systematic-debugging` AND `/no-workarounds` BEFORE CHANGING CODE OR TESTS
- FOLLOW THE PROJECT E2E PATTERN — use the existing daemon-served Playwright harness under `web/e2e/`; do not replace it with one-off browser scripts or Vite-only checks
- FOCUS ON SHIPPED OPERATOR FLOWS — settings shell navigation, restart-aware saves, collection CRUD, workspace-scoped MCP behavior, and hooks/extensions hybrid behavior all need durable proof
- DO NOT WEAKEN TESTS TO GET GREEN — fix production code or configuration at the source, then rerun the affected scenarios and full gates
- GREENFIELD: a cobertura E2E de settings precisa entrar no fluxo normal do projeto (`make test-e2e-web` / `make verify`), não ficar como verificação paralela e esquecível
</critical>

<requirements>
- MUST use the `/qa-execution` skill with `qa-output-path=.compozy/tasks/settings-ui`
- MUST consume `.compozy/tasks/settings-ui/qa/test-plans/` and `.compozy/tasks/settings-ui/qa/test-cases/` from `task_15` as the execution matrix seed
- MUST add committed daemon-served Playwright coverage for the settings feature under `web/e2e/`, following the existing shared fixture pattern
- MUST cover at least these critical flows in persistent E2E: settings shell navigation, one save flow with restart-required state, one collection CRUD flow, the workspace-scoped `mcp-servers` flow, and the `hooks-extensions` hybrid flow
- MUST write fresh QA evidence to `.compozy/tasks/settings-ui/qa/verification-report.md` and capture bugs/screenshots under the same artifact root
- MUST rerun the repository verification gates after the last fix, including the browser E2E lane that now contains the settings coverage
- SHOULD reuse `web/e2e/fixtures/test.ts`, `runtime.ts`, `runtime-seed.ts`, and `selectors.ts` rather than inventing a separate settings harness
</requirements>

## Design References

This task covers the complete settings surface and should treat all 10 Settings Paper artboards as browser-visible regression targets. Use `_techspec.md` → "Design References" plus the local exports in `docs/design/paper/settings/` when validating route state, section transitions, and UI expectations.

## Subtasks

- [ ] 16.1 Activate `/qa-execution` with `qa-output-path=.compozy/tasks/settings-ui` and derive the execution matrix from `task_15` artifacts
- [ ] 16.2 Extend shared Playwright selector/runtime-seed helpers for settings flows only where the existing `web/e2e` pattern needs explicit support
- [ ] 16.3 Implement one or more daemon-served browser E2E specs for the critical settings operator flows
- [ ] 16.4 Execute CLI, API, and browser QA flows, capture evidence/bugs, and fix root-cause regressions with matching regression tests
- [ ] 16.5 Rerun `make test-e2e-web`, `make verify`, and publish `.compozy/tasks/settings-ui/qa/verification-report.md`

## Implementation Details

See TechSpec sections "Testing Approach", "Web route coverage", "Verification gates", "Development Sequencing", and "Known Risks". The key constraint is that settings QA must become part of the repo's standard browser lane instead of a one-time exploratory run: committed Playwright scenarios should prove the operator-visible surface, while `/qa-execution` captures the broader execution report, screenshots, and issue documentation.

### Relevant Files

- `.agents/skills/qa-execution/SKILL.md` — required workflow for execution matrix discovery, evidence capture, and verification reporting
- `.compozy/tasks/settings-ui/qa/test-plans/` — planning artifacts that seed the QA execution matrix
- `.compozy/tasks/settings-ui/qa/test-cases/` — manual cases and priorities that the execution pass must honor
- `scripts/discover-project-contract.py` — canonical repo-contract discovery entrypoint required by `/qa-execution`
- `Makefile` — repository-defined `verify` and E2E entrypoints that must pass before completion
- `web/playwright.config.ts` — shared Playwright configuration for the daemon-served browser lane
- `web/e2e/fixtures/test.ts` — canonical browser fixture entrypoint used by repo E2E specs
- `web/e2e/fixtures/runtime.ts` — daemon-served runtime harness for browser E2E
- `web/e2e/fixtures/runtime-seed.ts` — seeded runtime helpers that settings flows should extend instead of duplicating
- `web/e2e/fixtures/selectors.ts` — shared selector helpers that settings coverage should expand if needed
- `web/src/routes/_app/settings/*.tsx` — shipped route surfaces that the new E2E specs must exercise
- `web/src/systems/settings/` — shared frontend domain that may need regression fixes surfaced by QA

### Dependent Files

- `web/e2e/settings.spec.ts` or `web/e2e/settings-*.spec.ts` — committed daemon-served settings browser E2E coverage
- `web/e2e/fixtures/selectors.ts` — may gain stable settings selectors for new browser flows
- `web/e2e/fixtures/runtime-seed.ts` — may gain deterministic settings seed helpers for collections, restart state, or workspace-scoped MCP data
- `.compozy/tasks/settings-ui/qa/verification-report.md` — final QA evidence written by `/qa-execution`
- `.compozy/tasks/settings-ui/qa/screenshots/` — browser evidence for the executed settings flows
- `.compozy/tasks/settings-ui/qa/issues/BUG-*.md` — structured bug reports for any failures discovered during execution
- `web/src/routes/_app/-settings*.test.tsx` and `web/src/systems/settings/**/*.test.ts` — narrow regression coverage that may need updates when QA finds root causes

### Related ADRs

- [ADR-001: Use a consolidated settings namespace with a dedicated settings shell](adrs/adr-001.md) — Browser E2E must validate settings as one navigable product surface
- [ADR-002: Persist settings by writing canonical config overlays instead of creating a new settings store](adrs/adr-002.md) — E2E must prove source precedence, write-target behavior, and workspace-scoped MCP flows
- [ADR-003: Keep settings mutations restart-aware and separate from operational workflows](adrs/adr-003.md) — Critical execution flows must distinguish restart-required saves from immediate operational actions
- [ADR-004: Restrict HTTP settings mutations to loopback-bound servers in v1](adrs/adr-004.md) — QA execution must validate operator-visible mutation restrictions and error messaging

## Deliverables

- Fresh `.compozy/tasks/settings-ui/qa/verification-report.md` produced by `/qa-execution`
- Committed daemon-served Playwright settings E2E coverage under `web/e2e/` **(REQUIRED)**
- Shared settings browser fixture or selector support only where needed by the new E2E specs **(REQUIRED)**
- Root-cause bug fixes plus matching regression tests for any issues discovered during execution **(REQUIRED)**
- Fresh screenshots and bug reports under `.compozy/tasks/settings-ui/qa/` for the executed settings flows **(REQUIRED)**
- Passing `make test-e2e-web` and `make verify` after the final QA fixes **(REQUIRED)**

## Tests

- Unit tests:
  - [ ] New settings browser selector helpers resolve stable shell, section, collection, and restart surfaces
  - [ ] Runtime seed helpers can create deterministic settings prerequisites without hidden global-state assumptions
- Integration tests:
  - [ ] Settings Playwright coverage exercises shell navigation plus a representative save flow with restart-required messaging and status polling
  - [ ] Settings Playwright coverage exercises at least one collection CRUD flow (`providers` or `environments`) through the daemon-served UI
  - [ ] Settings Playwright coverage exercises the workspace-scoped `mcp-servers` flow, including visible scope/target semantics
  - [ ] Settings Playwright coverage exercises the `hooks-extensions` page and distinguishes immediate operational actions from restart-aware config edits
  - [ ] `make test-e2e-web` passes with the settings scenarios included in the browser lane
  - [ ] `make verify` passes after the final QA fix set

## Success Criteria

- The `/qa-execution` workflow has been run explicitly with artifacts stored under `.compozy/tasks/settings-ui/qa/`
- The settings feature has committed browser E2E coverage that follows the repo's existing daemon-served Playwright pattern
- Any QA failures were fixed at the source and documented with fresh evidence
- The normal repo verification gates, including the browser E2E lane, pass with the new settings coverage in place
