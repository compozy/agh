---
status: completed
title: "Task Execution Profile Domain, Service Authority, and Store CRUD"
type: backend
complexity: high
dependencies:
  - task_02
  - task_03
---

# Task 08: Task Execution Profile Domain, Service Authority, and Store CRUD

## Overview
This task introduced the domain and authority layer for task execution profiles. It added typed profile validation, task-manager CRUD authority, active-run mutation rejection, audit events, and transactional GlobalDB persistence over selector tables.

<critical>
- ALWAYS READ `_techspec.md`, `_techspec_orchestration.md`, `_techspec_review_gate.md`, every ADR, and the dependency task files before starting.
- REFERENCE TECHSPEC for implementation details; do not duplicate architecture or code snippets here.
- FOCUS ON WHAT needs to be delivered, keep changes scoped, and avoid compatibility shims or fallback paths.
- TESTS REQUIRED: every production change must ship focused tests and run the listed verification gates.
</critical>

<requirements>
- MUST model execution profiles with typed domain structs and bounded validation.
- MUST keep task manager as the write authority for profile CRUD.
- MUST persist selector tables atomically without metadata escape hatches.
</requirements>

## Subtasks
- [x] Add `task.ExecutionProfile` and selector value types.
- [x] Implement task manager get/set/delete profile authority.
- [x] Persist profiles and selector rows transactionally in GlobalDB.
- [x] Add validation, manager, and store tests.

## Implementation Details
Required skill activation must match the touched surfaces: backend tasks use `agh-code-guidelines`, `golang-pro`, and `agh-test-conventions`; contract tasks also use `agh-contract-codegen-coship`; web tasks use the web instructions and frontend/design skills; docs tasks use `documentation-writer`, `copywriting` when public prose changes, and the site instructions; QA tasks use the QA skills named in the task. Use the TechSpec and ADRs for architecture; this task records scope and evidence boundaries.

### Relevant Files
- `internal/task/profile.go` - execution profile domain.
- `internal/task/manager_profile.go` - service authority.
- `internal/store/globaldb/global_db_task_profile.go` - profile persistence.

### Dependent Files
- `internal/task/profile_test.go` - validation tests.
- `internal/task/manager_profile_test.go` - authority tests.
- `internal/store/globaldb/global_db_task_profile_test.go` - persistence tests.

### Related ADRs
- [ADR-010: Typed Overlay](adrs/adr-010.md) - typed orchestration/profile overlays instead of metadata JSON.

## Extensibility / Agent Manageability / Config Lifecycle
- Extensibility: Execution profiles become a typed extension point for coordinator/worker/reviewer/sandbox selection.
- Agent manageability: Native tools and CLI/HTTP/UDS surfaces arrive in tasks 16 and 18.
- Config lifecycle: Consumes task orchestration config defaults from task 01.

### Web/Docs Impact
- `web/`: Profile editor UI planned in tasks 26 and 27.
- `packages/site`: Profile docs planned in task 28.

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
- State: `state.yaml.progress.checklist` iteration 17 is `completed`.
- Memory: `memory/free-iter-016.md`.
- Verification: the workflow memory records final `make verify` PASS for this slice.

## Success Criteria
- All tests passing.
- Test coverage >=80% for changed code paths, or 100% documented evidence coverage for docs-only tasks.
- `make verify` passes before the task is marked complete.
- The task evidence is recorded in workflow memory or QA artifacts.
