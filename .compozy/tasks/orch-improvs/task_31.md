---
status: completed
title: "QA Plan and Test Coverage"
type: test
complexity: high
dependencies:
  - task_30
---

# Task 31: QA Plan and Test Coverage

## Overview
This task produces the mandatory QA report for the full orchestration-improvements program. It must plan behavior-first coverage across backend runtime, CLI/HTTP/UDS, native tools, web UI, site docs, config lifecycle, migrations, review routing, notification cursors, and lessons.

<critical>
- ALWAYS READ `_techspec.md`, `_techspec_orchestration.md`, `_techspec_review_gate.md`, every ADR, and the dependency task files before starting.
- REFERENCE TECHSPEC for implementation details; do not duplicate architecture or code snippets here.
- FOCUS ON WHAT needs to be delivered, keep changes scoped, and avoid compatibility shims or fallback paths.
- TESTS REQUIRED: every production change must ship focused tests and run the listed verification gates.
</critical>

<requirements>
- MUST activate `qa-report` and produce QA planning artifacts under `.compozy/tasks/orch-improvs/qa/`.
- MUST cover every public surface touched by tasks 01 through 30.
- MUST include real-scenario, e2e, negative, concurrency, migration, and contract drift cases.
</requirements>

## Subtasks
- [x] Read all TechSpecs, ADRs, task files, workflow memory, and completed evidence.
- [x] Create QA plan and test cases covering CLI, HTTP, UDS, native tools, web, site docs, migrations, and review flows.
- [x] Identify high-risk regression hot spots and required gates.
- [x] Persist QA artifacts under `qa/` with clear filenames.
- [x] Update managed state only after artifacts exist.

## Implementation Details
Required skill activation must match the touched surfaces: backend tasks use `agh-code-guidelines`, `golang-pro`, and `agh-test-conventions`; contract tasks also use `agh-contract-codegen-coship`; web tasks use the web instructions and frontend/design skills; docs tasks use `documentation-writer`, `copywriting` when public prose changes, and the site instructions; QA tasks use the QA skills named in the task. Use the TechSpec and ADRs for architecture; this task records scope and evidence boundaries.

### Relevant Files
- `.compozy/tasks/orch-improvs/qa` - QA artifact destination.
- `.compozy/tasks/orch-improvs/_tasks.md` - coverage source.
- `.compozy/tasks/orch-improvs/memory` - implementation evidence.
- `web/e2e` - UI e2e planning target.

### Dependent Files
- `make verify` - full monorepo gate.
- `make test-e2e-runtime` - runtime e2e gate.
- `make test-e2e-web` - UI e2e gate.

### Related ADRs
- [ADR-003: Durable Cursor Primitive](adrs/adr-003.md) - notification cursor and replay semantics.
- [ADR-005: Denormalized Current Run Projection](adrs/adr-005.md) - current run projection boundaries.
- [ADR-007: Post-Terminal Review Gate](adrs/adr-007.md) - review request/verdict/continuation authority.
- [ADR-010: Typed Overlay](adrs/adr-010.md) - execution profile schema and config overlay shape.

## Extensibility / Agent Manageability / Config Lifecycle
- Extensibility: QA plan must include extension, skills, tools, bridge, and agent-manageability paths.
- Agent manageability: QA plan must compare CLI/HTTP/UDS/native/web behavior for the same persisted state.
- Config lifecycle: QA plan must verify config defaults, overlays, validation, and docs examples.

### Web/Docs Impact
- `web/`: Plan Playwright/browser-use coverage for task profile/review/notification UI.
- `packages/site`: Plan site docs checks and generated reference consistency.

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
- Authored the mandatory QA report artifacts under `.compozy/tasks/orch-improvs/qa/`:
  - `qa/test-plans/orch-improvs-test-plan.md`
  - `qa/test-plans/orch-improvs-regression-suite.md`
  - `qa/test-cases/TC-SCEN-001-full-orchestration-review-loop.md`
  - `qa/test-cases/TC-INT-001-config-schema-migration-and-profile-parity.md`
  - `qa/test-cases/TC-INT-002-review-gate-contract-and-continuation.md`
  - `qa/test-cases/TC-INT-003-notification-cursor-and-bridge-delivery.md`
  - `qa/test-cases/TC-UI-001-web-orchestration-tab-operator-truth.md`
  - `qa/test-cases/TC-REG-001-generated-contracts-cli-site-docs-drift.md`
  - `qa/test-cases/TC-SEC-001-claim-token-redaction-and-reviewer-boundary.md`
  - `qa/test-cases/TC-PERF-001-sse-query-churn-and-cursor-replay.md`
  - `qa/issues/README.md`
  - `qa/screenshots/README.md`
- The plan covers all public surfaces touched by tasks 01-30: config, GlobalDB migrations, task execution profiles, review routing/verdicts/continuations, bundled skills, native tools, HTTP, UDS, CLI, OpenAPI/codegen, web data/UI, task context bundle, SSE replay, notification cursors, bridge subscriptions, site docs, generated CLI references, lessons, and glossary.
- The cases include real-scenario, e2e, negative, concurrency, migration, security/redaction, performance, and contract drift coverage. Smoke readiness is explicitly separated from release-grade behavioral evidence.
- `qa/issues/` and `qa/screenshots/` are intentionally empty except README handoff files; `task_32` owns execution bugs and browser artifacts.

## Verification Evidence
- `compozy tasks validate --name orch-improvs --format json` PASS, `scanned: 32`.
- `git diff --check` PASS.
- QA artifact structural checks PASS: every `TC-*.md` case includes priority, objective, preconditions, test steps with `Expected`, behavioral evidence, and disruption probes.
- QA artifact marker scan PASS: no unfinished-work markers in `qa/`.
- `make verify` PASS. Evidence: Bun/Vitest monorepo `339 passed (339)` files / `2206 passed (2206)` tests, web build PASS, `golangci-lint` `0 issues`, Go race gate `DONE 8283 tests in 11.125s`, package boundaries `OK`.

## Success Criteria
- All tests passing.
- Test coverage >=80% for changed code paths, or 100% documented evidence coverage for docs-only tasks.
- `make verify` passes before the task is marked complete.
- The task evidence is recorded in workflow memory or QA artifacts.
