---
status: completed
title: "Run Review Request, Binding, and Review Store Authority"
type: backend
complexity: high
dependencies:
  - task_04
  - task_08
---

# Task 09: Run Review Request, Binding, and Review Store Authority

## Overview
This task added review request and reviewer-session binding authority. It introduced typed review rows, status/outcome/policy validation, idempotent request creation, active reviewer binding, lookup, and listing behavior.

<critical>
- ALWAYS READ `_techspec.md`, `_techspec_orchestration.md`, `_techspec_review_gate.md`, every ADR, and the dependency task files before starting.
- REFERENCE TECHSPEC for implementation details; do not duplicate architecture or code snippets here.
- FOCUS ON WHAT needs to be delivered, keep changes scoped, and avoid compatibility shims or fallback paths.
- TESTS REQUIRED: every production change must ship focused tests and run the listed verification gates.
</critical>

<requirements>
- MUST create review requests idempotently for a run/round/attempt.
- MUST bind reviewer sessions transactionally and expose session lookup.
- MUST keep review request/list authority in `task.Service`.
</requirements>

## Subtasks
- [x] Add run-review domain types and validation.
- [x] Implement request, bind, lookup, and list methods on task manager.
- [x] Persist review requests and reviewer bindings in GlobalDB.
- [x] Update fakes/stubs and focused review tests.

## Implementation Details
Required skill activation must match the touched surfaces: backend tasks use `agh-code-guidelines`, `golang-pro`, and `agh-test-conventions`; contract tasks also use `agh-contract-codegen-coship`; web tasks use the web instructions and frontend/design skills; docs tasks use `documentation-writer`, `copywriting` when public prose changes, and the site instructions; QA tasks use the QA skills named in the task. Use the TechSpec and ADRs for architecture; this task records scope and evidence boundaries.

### Relevant Files
- `internal/task/review.go` - review domain.
- `internal/task/manager_review.go` - service authority.
- `internal/store/globaldb/global_db_task_review.go` - review persistence.

### Dependent Files
- `internal/task/review_test.go` - validation tests.
- `internal/task/manager_review_test.go` - manager tests.
- `internal/store/globaldb/global_db_task_review_test.go` - store tests.

### Related ADRs
- [ADR-007: Post-Terminal Review Gate](adrs/adr-007.md) - reviews happen after terminal runs and continuations are explicit runs.

## Extensibility / Agent Manageability / Config Lifecycle
- Extensibility: Reviewer sessions become a first-class runtime authority surface.
- Agent manageability: Native and transport tools expose this in tasks 15, 17, and 19.
- Config lifecycle: Consumes review policy from typed config/profile state.

### Web/Docs Impact
- `web/`: Review queue/read-model UI planned in tasks 26 and 27.
- `packages/site`: Review-gate docs planned in task 29.

## Deliverables
- Task implementation or documentation matching the requirements above.
- Focused unit tests with 80%+ coverage where code changes.
- Integration, contract, e2e, or docs-build tests proportional to the touched behavior.
- Updated workflow memory, QA evidence, generated artifacts, or site docs when applicable.

## Tests
- Unit tests:
  - [x] Validate the primary success path for this task.
  - [x] Validate malformed input, missing dependency, or authorization failure paths.
  - [x] Validate boundary conditions named by the related TechSpec and ADRs.
- Integration tests:
  - [x] Exercise the task through the owning service/transport boundary when applicable.
  - [x] Compare persisted state, generated contract output, or rendered docs/UI with runtime truth.
  - [x] Run race, codegen, site, web, or full verify gates listed by the touched surface.
- Test coverage target: >=80% for changed code paths; docs-only tasks require 100% checklist evidence against authored pages.
- All tests must pass.

## Completion Evidence
- State: `state.yaml.progress.checklist` iteration 19 is `completed`.
- Memory: `memory/free-iter-018.md`.
- Verification: the workflow memory records final `make verify` PASS for this slice.

## Success Criteria
- All tests passing.
- Test coverage >=80% for changed code paths, or 100% documented evidence coverage for docs-only tasks.
- `make verify` passes before the task is marked complete.
- The task evidence is recorded in workflow memory or QA artifacts.
