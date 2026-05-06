---
status: completed
title: "Task Orchestration Config Defaults and Validation"
type: backend
complexity: medium
dependencies: []
---

# Task 01: Task Orchestration Config Defaults and Validation

## Overview
This task established the typed configuration foundation for task orchestration profiles and review policy. It made orchestration settings explicit in config structs, defaults, validation, and agent-mutable config paths so later runtime behavior could consume a stable contract.

<critical>
- ALWAYS READ `_techspec.md`, `_techspec_orchestration.md`, `_techspec_review_gate.md`, every ADR, and the dependency task files before starting.
- REFERENCE TECHSPEC for implementation details; do not duplicate architecture or code snippets here.
- FOCUS ON WHAT needs to be delivered, keep changes scoped, and avoid compatibility shims or fallback paths.
- TESTS REQUIRED: every production change must ship focused tests and run the listed verification gates.
</critical>

<requirements>
- MUST define typed `[task.orchestration]`, profile, and review config defaults.
- MUST validate invalid profile/review values before runtime use.
- MUST expose only the intended config paths as agent-mutable.
</requirements>

## Subtasks
- [x] Add typed config structs and defaults for task orchestration.
- [x] Wire TOML merge/overlay handling and validation.
- [x] Register agent-mutable paths for orchestration settings.
- [x] Cover valid, invalid, default, and mutable-path behavior with tests.

## Implementation Details
Required skill activation must match the touched surfaces: backend tasks use `agh-code-guidelines`, `golang-pro`, and `agh-test-conventions`; contract tasks also use `agh-contract-codegen-coship`; web tasks use the web instructions and frontend/design skills; docs tasks use `documentation-writer`, `copywriting` when public prose changes, and the site instructions; QA tasks use the QA skills named in the task. Use the TechSpec and ADRs for architecture; this task records scope and evidence boundaries.

### Relevant Files
- `internal/config/task_orchestration.go` - typed orchestration config model.
- `internal/config/config.go` - root config composition.
- `internal/config/merge.go` - TOML overlay behavior.
- `internal/config/tool_surface.go` - agent-mutable config path registry.

### Dependent Files
- `internal/config/task_orchestration_test.go` - default and validation coverage.
- `internal/config/tool_surface_test.go` - mutable path coverage.

### Related ADRs
- [ADR-010: Typed Overlay](adrs/adr-010.md) - typed orchestration/profile overlays instead of metadata JSON.

## Extensibility / Agent Manageability / Config Lifecycle
- Extensibility: Introduced typed runtime config consumed by profile and review services.
- Agent manageability: Agents can manage the safe config surface through the mutable config path registry.
- Config lifecycle: Added and validated `[task.orchestration]`, `[task.orchestration.profile]`, and `[task.orchestration.review]` defaults.

### Web/Docs Impact
- `web/`: No direct `web/` change in this slice; later contract/UI tasks consume the behavior.
- `packages/site`: Narrative docs are deferred to tasks 28 and 29.

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
- State: `state.yaml.progress.checklist` iteration 3 is `completed`.
- Memory: `memory/free-iter-002.md`.
- Verification: the workflow memory records final `make verify` PASS for this slice.

## Success Criteria
- All tests passing.
- Test coverage >=80% for changed code paths, or 100% documented evidence coverage for docs-only tasks.
- `make verify` passes before the task is marked complete.
- The task evidence is recorded in workflow memory or QA artifacts.
