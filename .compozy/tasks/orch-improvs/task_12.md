---
status: completed
title: "Worker Agent, Provider, and Model Session Selection"
type: backend
complexity: high
dependencies:
  - task_08
---

# Task 12: Worker Agent, Provider, and Model Session Selection

## Overview
This task applied execution-profile worker runtime selection when starting task sessions. It validates provider/model overrides through config resolution and passes worker agent/provider/model into daemon session creation.

<critical>
- ALWAYS READ `_techspec.md`, `_techspec_orchestration.md`, `_techspec_review_gate.md`, every ADR, and the dependency task files before starting.
- REFERENCE TECHSPEC for implementation details; do not duplicate architecture or code snippets here.
- FOCUS ON WHAT needs to be delivered, keep changes scoped, and avoid compatibility shims or fallback paths.
- TESTS REQUIRED: every production change must ship focused tests and run the listed verification gates.
</critical>

<requirements>
- MUST load the effective execution profile before task session start.
- MUST validate runtime provider/model overrides through provider config.
- MUST pass worker agent, provider, and model into session creation.
</requirements>

## Subtasks
- [x] Load execution profiles during task run session startup.
- [x] Add runtime model/provider resolution in config/session.
- [x] Map worker runtime settings in daemon task session bridge.
- [x] Add config, session, task, and daemon tests.

## Implementation Details
Required skill activation must match the touched surfaces: backend tasks use `agh-code-guidelines`, `golang-pro`, and `agh-test-conventions`; contract tasks also use `agh-contract-codegen-coship`; web tasks use the web instructions and frontend/design skills; docs tasks use `documentation-writer`, `copywriting` when public prose changes, and the site instructions; QA tasks use the QA skills named in the task. Use the TechSpec and ADRs for architecture; this task records scope and evidence boundaries.

### Relevant Files
- `internal/task/manager.go` - task session start flow.
- `internal/daemon/task_runtime.go` - daemon session bridge.
- `internal/config` - runtime provider/model resolution.
- `internal/session/manager.go` - create options.

### Dependent Files
- `internal/config` tests - provider/model override behavior.
- `internal/session` tests - create option behavior.
- `internal/daemon` tests - task runtime handoff.

### Related ADRs
- [ADR-010: Typed Overlay](adrs/adr-010.md) - typed orchestration/profile overlays instead of metadata JSON.

## Extensibility / Agent Manageability / Config Lifecycle
- Extensibility: Execution profiles select runtime agent/provider/model for task workers.
- Agent manageability: Agents can inspect/update profile choices through later surfaces.
- Config lifecycle: Ensures profile overrides remain constrained by configured providers/models.

### Web/Docs Impact
- `web/`: Worker runtime choices surface in tasks 26 and 27.
- `packages/site`: Runtime profile docs planned in task 28.

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
- State: `state.yaml.progress.checklist` iteration 25 is `completed`.
- Memory: `memory/free-iter-024.md`.
- Verification: the workflow memory records final `make verify` PASS for this slice.

## Success Criteria
- All tests passing.
- Test coverage >=80% for changed code paths, or 100% documented evidence coverage for docs-only tasks.
- `make verify` passes before the task is marked complete.
- The task evidence is recorded in workflow memory or QA artifacts.
