---
status: completed
title: "Native Task Execution Profile Tools"
type: backend
complexity: medium
dependencies:
  - task_08
---

# Task 16: Native Task Execution Profile Tools

## Overview
This task added native task execution profile get, set, and delete tools. They delegate to task-service profile authority and reject malformed or server-owned fields before writes.

<critical>
- ALWAYS READ `_techspec.md`, `_techspec_orchestration.md`, `_techspec_review_gate.md`, every ADR, and the dependency task files before starting.
- REFERENCE TECHSPEC for implementation details; do not duplicate architecture or code snippets here.
- FOCUS ON WHAT needs to be delivered, keep changes scoped, and avoid compatibility shims or fallback paths.
- TESTS REQUIRED: every production change must ship focused tests and run the listed verification gates.
</critical>

<requirements>
- MUST expose native profile get/set/delete tools through the task toolset.
- MUST reject server-owned and unknown profile fields before mutation.
- MUST delegate profile writes to `task.Service` authority.
</requirements>

## Subtasks
- [x] Add builtin IDs and descriptors for profile tools.
- [x] Register tools in the task toolset.
- [x] Implement daemon native profile bindings.
- [x] Add descriptor, routing, and invalid-payload tests.

## Implementation Details
Required skill activation must match the touched surfaces: backend tasks use `agh-code-guidelines`, `golang-pro`, and `agh-test-conventions`; contract tasks also use `agh-contract-codegen-coship`; web tasks use the web instructions and frontend/design skills; docs tasks use `documentation-writer`, `copywriting` when public prose changes, and the site instructions; QA tasks use the QA skills named in the task. Use the TechSpec and ADRs for architecture; this task records scope and evidence boundaries.

### Relevant Files
- `internal/tools/builtin/tasks.go` - task tool descriptors.
- `internal/daemon/native_profile_tools.go` - native bindings.
- `internal/daemon/native_tools.go` - registry wiring.

### Dependent Files
- `internal/tools/builtin/builtin_test.go` - descriptor tests.
- `internal/daemon/native_tools_test.go` - native profile tests.

### Related ADRs
- [ADR-010: Typed Overlay](adrs/adr-010.md) - typed orchestration/profile overlays instead of metadata JSON.

## Extensibility / Agent Manageability / Config Lifecycle
- Extensibility: Adds model-facing profile management tools.
- Agent manageability: HTTP/UDS/CLI profile management is handled in task 18.
- Config lifecycle: Profiles remain constrained by typed validation/defaults.

### Web/Docs Impact
- `web/`: Profile editor UI planned in task 27.
- `packages/site`: Profile tool docs planned in task 28.

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
- State: `state.yaml.progress.checklist` iteration 33 is `completed`.
- Memory: `memory/free-iter-032.md`.
- Verification: the workflow memory records final `make verify` PASS for this slice.

## Success Criteria
- All tests passing.
- Test coverage >=80% for changed code paths, or 100% documented evidence coverage for docs-only tasks.
- `make verify` passes before the task is marked complete.
- The task evidence is recorded in workflow memory or QA artifacts.
