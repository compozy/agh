---
status: completed
title: "Sandbox Profile Runtime Application"
type: backend
complexity: high
dependencies:
  - task_08
  - task_12
---

# Task 13: Sandbox Profile Runtime Application

## Overview
This task applied execution-profile sandbox mode at task session start. It supports inherit, none, and explicit sandbox ref while preserving workspace sandbox authorization and avoiding implicit cleanup behavior for no-sandbox sessions.

<critical>
- ALWAYS READ `_techspec.md`, `_techspec_orchestration.md`, `_techspec_review_gate.md`, every ADR, and the dependency task files before starting.
- REFERENCE TECHSPEC for implementation details; do not duplicate architecture or code snippets here.
- FOCUS ON WHAT needs to be delivered, keep changes scoped, and avoid compatibility shims or fallback paths.
- TESTS REQUIRED: every production change must ship focused tests and run the listed verification gates.
</critical>

<requirements>
- MUST map profile sandbox modes to session create options.
- MUST resolve explicit sandbox refs through workspace authorization.
- MUST preserve no-sandbox semantics without inheriting sandbox cleanup flags.
</requirements>

## Subtasks
- [x] Add sandbox ref and disable-sandbox create options.
- [x] Map profile sandbox policy in daemon task session bridge.
- [x] Resolve explicit sandbox refs through workspace config.
- [x] Add session and daemon sandbox tests.

## Implementation Details
Required skill activation must match the touched surfaces: backend tasks use `agh-code-guidelines`, `golang-pro`, and `agh-test-conventions`; contract tasks also use `agh-contract-codegen-coship`; web tasks use the web instructions and frontend/design skills; docs tasks use `documentation-writer`, `copywriting` when public prose changes, and the site instructions; QA tasks use the QA skills named in the task. Use the TechSpec and ADRs for architecture; this task records scope and evidence boundaries.

### Relevant Files
- `internal/daemon/task_runtime.go` - profile sandbox mapping.
- `internal/session/manager.go` - create options.
- `internal/session/manager_workspace.go` - workspace sandbox resolution.
- `internal/session/sandbox.go` - no-sandbox setup.

### Dependent Files
- `internal/session/manager_sandbox_test.go` - sandbox override tests.
- `internal/daemon/task_runtime_test.go` - bridge runtime tests.

### Related ADRs
- [ADR-010: Typed Overlay](adrs/adr-010.md) - typed orchestration/profile overlays instead of metadata JSON.

## Extensibility / Agent Manageability / Config Lifecycle
- Extensibility: Profiles control sandbox policy through the runtime session boundary.
- Agent manageability: Agents manage sandbox choices through profile surfaces.
- Config lifecycle: Explicit sandbox refs are authorized by existing workspace config.

### Web/Docs Impact
- `web/`: Sandbox profile editor field planned in task 27.
- `packages/site`: Sandbox profile documentation planned in task 28.

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
- State: `state.yaml.progress.checklist` iteration 27 is `completed`.
- Memory: `memory/free-iter-026.md`.
- Verification: the workflow memory records final `make verify` PASS for this slice.

## Success Criteria
- All tests passing.
- Test coverage >=80% for changed code paths, or 100% documented evidence coverage for docs-only tasks.
- `make verify` passes before the task is marked complete.
- The task evidence is recorded in workflow memory or QA artifacts.
