---
status: completed
title: "Current Run Projection and Transition Invariants"
type: backend
complexity: high
dependencies:
  - task_02
---

# Task 05: Current Run Projection and Transition Invariants

## Overview
This task made `tasks.current_run_id` a denormalized read model owned by GlobalDB transitions. It ensured claims, lease release, terminal transitions, recovery, and service-managed run updates keep the projection coherent.

<critical>
- ALWAYS READ `_techspec.md`, `_techspec_orchestration.md`, `_techspec_review_gate.md`, every ADR, and the dependency task files before starting.
- REFERENCE TECHSPEC for implementation details; do not duplicate architecture or code snippets here.
- FOCUS ON WHAT needs to be delivered, keep changes scoped, and avoid compatibility shims or fallback paths.
- TESTS REQUIRED: every production change must ship focused tests and run the listed verification gates.
</critical>

<requirements>
- MUST update current-run projection in the same transaction as run transitions.
- MUST clear projection on terminal, release, and recovery states where no run remains active.
- MUST prevent external mutators from owning the projection.
</requirements>

## Subtasks
- [x] Add `CurrentRunID` to task domain and read models.
- [x] Maintain projection in claim, release, terminal, and recovery transactions.
- [x] Update service-managed run transitions to keep projection coherent.
- [x] Add projection invariants to GlobalDB tests.

## Implementation Details
Required skill activation must match the touched surfaces: backend tasks use `agh-code-guidelines`, `golang-pro`, and `agh-test-conventions`; contract tasks also use `agh-contract-codegen-coship`; web tasks use the web instructions and frontend/design skills; docs tasks use `documentation-writer`, `copywriting` when public prose changes, and the site instructions; QA tasks use the QA skills named in the task. Use the TechSpec and ADRs for architecture; this task records scope and evidence boundaries.

### Relevant Files
- `internal/store/globaldb/global_db_task_projection.go` - projection maintenance.
- `internal/store/globaldb/global_db_task_claim.go` - claim transition integration.
- `internal/task/types.go` - domain/read model shape.

### Dependent Files
- `internal/store/globaldb/global_db_task_claim_test.go` - projection tests.
- `internal/store/globaldb/global_db_task_test.go` - read-model equality.

### Related ADRs
- [ADR-005: Denormalized Current Run Projection](adrs/adr-005.md) - current run read model is owned by GlobalDB transitions.

## Extensibility / Agent Manageability / Config Lifecycle
- Extensibility: Created a stable projection consumed by web/API/task summaries.
- Agent manageability: Agents later consume this through task summary and context-bundle surfaces.
- Config lifecycle: No config changes.

### Web/Docs Impact
- `web/`: Impacts task summary UI and query models planned in tasks 26 and 27.
- `packages/site`: Projection behavior documented in task 28/29 docs.

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
- State: `state.yaml.progress.checklist` iteration 11 is `completed`.
- Memory: `memory/free-iter-010.md`.
- Verification: the workflow memory records final `make verify` PASS for this slice.

## Success Criteria
- All tests passing.
- Test coverage >=80% for changed code paths, or 100% documented evidence coverage for docs-only tasks.
- `make verify` passes before the task is marked complete.
- The task evidence is recorded in workflow memory or QA artifacts.
