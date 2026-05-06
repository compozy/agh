---
status: completed
title: "Run Review Verdicts, Continuation Runs, and Review Events"
type: backend
complexity: critical
dependencies:
  - task_09
---

# Task 10: Run Review Verdicts, Continuation Runs, and Review Events

## Overview
This task completed review verdict authority and continuation run creation. It records approved, rejected, and blocked outcomes with delivery-id replay safety and creates rejected-review continuation runs atomically.

<critical>
- ALWAYS READ `_techspec.md`, `_techspec_orchestration.md`, `_techspec_review_gate.md`, every ADR, and the dependency task files before starting.
- REFERENCE TECHSPEC for implementation details; do not duplicate architecture or code snippets here.
- FOCUS ON WHAT needs to be delivered, keep changes scoped, and avoid compatibility shims or fallback paths.
- TESTS REQUIRED: every production change must ship focused tests and run the listed verification gates.
</critical>

<requirements>
- MUST enforce reviewer-session binding before recording a verdict.
- MUST make verdict replay idempotent by delivery id.
- MUST create rejected-review continuation runs atomically with verdict persistence.
</requirements>

## Subtasks
- [x] Add verdict/result domain types and validation.
- [x] Implement task manager `RecordRunReview` authority and events.
- [x] Persist verdicts, task rollups, and continuation runs transactionally.
- [x] Add approved, rejected, blocked, and replay tests.

## Implementation Details
Required skill activation must match the touched surfaces: backend tasks use `agh-code-guidelines`, `golang-pro`, and `agh-test-conventions`; contract tasks also use `agh-contract-codegen-coship`; web tasks use the web instructions and frontend/design skills; docs tasks use `documentation-writer`, `copywriting` when public prose changes, and the site instructions; QA tasks use the QA skills named in the task. Use the TechSpec and ADRs for architecture; this task records scope and evidence boundaries.

### Relevant Files
- `internal/task/review.go` - verdict request/result types.
- `internal/task/manager_review.go` - verdict authority.
- `internal/store/globaldb/global_db_task_review.go` - verdict and continuation transaction.

### Dependent Files
- `internal/task/manager_review_test.go` - authority event tests.
- `internal/store/globaldb/global_db_task_review_test.go` - persistence/replay tests.

### Related ADRs
- [ADR-007: Post-Terminal Review Gate](adrs/adr-007.md) - reviews happen after terminal runs and continuations are explicit runs.

## Extensibility / Agent Manageability / Config Lifecycle
- Extensibility: Defines the runtime authority used by reviewer skills and native tools.
- Agent manageability: Reviewer-bound `submit_run_review` and transport verdict submission arrive in tasks 15 and 19.
- Config lifecycle: Uses review profile/policy for later routing tasks.

### Web/Docs Impact
- `web/`: Review verdict UX planned in task 27.
- `packages/site`: Review continuation behavior documented in task 29.

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
- State: `state.yaml.progress.checklist` iteration 21 is `completed`.
- Memory: `memory/free-iter-020.md`.
- Verification: the workflow memory records final `make verify` PASS for this slice.

## Success Criteria
- All tests passing.
- Test coverage >=80% for changed code paths, or 100% documented evidence coverage for docs-only tasks.
- `make verify` passes before the task is marked complete.
- The task evidence is recorded in workflow memory or QA artifacts.
