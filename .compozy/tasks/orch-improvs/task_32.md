---
status: completed
title: "Real-Scenario QA Execution"
type: test
complexity: critical
dependencies:
  - task_31
---

# Task 32: Real-Scenario QA Execution

## Overview
This task executes the mandatory real-scenario QA pass. It must use an isolated AGH lab, exercise runtime and web journeys end-to-end, record reproducible bugs, fix root causes, rerun gates, and update QA state only when evidence is complete.

<critical>
- ALWAYS READ `_techspec.md`, `_techspec_orchestration.md`, `_techspec_review_gate.md`, every ADR, and the dependency task files before starting.
- REFERENCE TECHSPEC for implementation details; do not duplicate architecture or code snippets here.
- FOCUS ON WHAT needs to be delivered, keep changes scoped, and avoid compatibility shims or fallback paths.
- TESTS REQUIRED: every production change must ship focused tests and run the listed verification gates.
</critical>

<requirements>
- MUST activate `agh-qa-bootstrap`, `real-scenario-qa`, `qa-execution`, and `agh-worktree-isolation`.
- MUST run isolated runtime, CLI/API/UDS/native-tool, web UI, and docs verification using the QA plan.
- MUST persist bootstrap manifest, lab paths, bug reports, and final verification report.
</requirements>

## Subtasks
- [x] Create or reuse an appropriate isolated QA bootstrap manifest for this exact QA pass.
- [x] Run CLI/HTTP/UDS/native review/profile/notification scenarios against real persisted state.
- [x] Run web e2e for profile editor, review queue/verdict, continuation, and notification diagnostics.
- [x] Run docs/site verification and full monorepo gates.
- [x] File and fix reproduced bugs, rerun evidence, and update managed state only when complete.

## Implementation Details
Required skill activation must match the touched surfaces: backend tasks use `agh-code-guidelines`, `golang-pro`, and `agh-test-conventions`; contract tasks also use `agh-contract-codegen-coship`; web tasks use the web instructions and frontend/design skills; docs tasks use `documentation-writer`, `copywriting` when public prose changes, and the site instructions; QA tasks use the QA skills named in the task. Use the TechSpec and ADRs for architecture; this task records scope and evidence boundaries.

### Relevant Files
- `.compozy/tasks/orch-improvs/qa` - execution artifacts and verification report.
- `web/e2e` - Playwright scenarios.
- `cmd/agh` and `internal/api` - runtime under test.
- `packages/site` - docs/site verification.

### Dependent Files
- `AGH_HOME` isolated QA lab paths - runtime state under test.
- `bootstrap-manifest.json` - resumable QA metadata.
- `make test-e2e-runtime` and `make test-e2e-web` - e2e gates.
- `make verify` - final monorepo gate.

### Related ADRs
- [ADR-003: Durable Cursor Primitive](adrs/adr-003.md) - notification cursor and replay semantics.
- [ADR-005: Denormalized Current Run Projection](adrs/adr-005.md) - current run projection boundaries.
- [ADR-007: Post-Terminal Review Gate](adrs/adr-007.md) - review request/verdict/continuation authority.
- [ADR-010: Typed Overlay](adrs/adr-010.md) - execution profile schema and config overlay shape.

## Extensibility / Agent Manageability / Config Lifecycle
- Extensibility: Proves extensibility surfaces operate together in a realistic runtime.
- Agent manageability: Proves agents/operators can manage profiles, reviews, notifications, and streams across CLI/HTTP/UDS/native/web.
- Config lifecycle: Proves config lifecycle defaults and examples work in a fresh isolated lab.

### Web/Docs Impact
- `web/`: Runs browser e2e for UI-bearing features via `browser-use:browser` with fallback only if unavailable.
- `packages/site`: Runs site build/typecheck and validates docs/reference coherence.

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
- QA bootstrap manifest persisted at `qa/bootstrap-manifest.json` with isolated `AGH_HOME`, UDS socket, web proxy target, provider homes, lab root, and artifact paths.
- Runtime CLI/HTTP/UDS/native-tool evidence persisted under `qa/evidence/runtime/`, including task/profile/review/notification/SSE flows and blocked live-provider boundaries.
- Browser evidence persisted under `qa/evidence/web/`; full daemon-served Playwright gate passed with 20 tests in `qa/evidence/gates/make-test-e2e-web-after-fixes.txt`.
- Docs evidence persisted under `qa/evidence/docs/`; runtime autonomy docs Vitest and full site validation passed.
- Bug reports `BUG-001` through `BUG-008` were filed under `qa/issues/` and fixed with focused tests plus E2E/gate reruns.
- Final verification report written to `qa/verification-report.md`.
- `make test-e2e-runtime` passed across `internal/daemon`, `internal/api/httpapi`, `internal/api/udsapi`, and `internal/testutil/e2e`; evidence is `qa/evidence/gates/make-test-e2e-runtime-final-pass.txt`.
- Final `make verify` passed with Bun/Vitest 339 files / 2206 tests, web build PASS, `golangci-lint` 0 issues, Go race gate `DONE 8290 tests in 98.066s`, and package boundaries OK; evidence is `qa/evidence/gates/make-verify-final.txt`.

## Success Criteria
- All tests passing.
- Test coverage >=80% for changed code paths, or 100% documented evidence coverage for docs-only tasks.
- `make verify` passes before the task is marked complete.
- The task evidence is recorded in workflow memory or QA artifacts.
