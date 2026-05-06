---
status: completed
title: "Native Run Review Request/List/Show Tools"
type: backend
complexity: medium
dependencies:
  - task_09
---

# Task 17: Native Run Review Request/List/Show Tools

## Overview
This task added native tools for review request, list, and show operations. The tools expose the review authority to model-facing sessions while keeping request/list/show behavior delegated to `task.Service`.

<critical>
- ALWAYS READ `_techspec.md`, `_techspec_orchestration.md`, `_techspec_review_gate.md`, every ADR, and the dependency task files before starting.
- REFERENCE TECHSPEC for implementation details; do not duplicate architecture or code snippets here.
- FOCUS ON WHAT needs to be delivered, keep changes scoped, and avoid compatibility shims or fallback paths.
- TESTS REQUIRED: every production change must ship focused tests and run the listed verification gates.
</critical>

<requirements>
- MUST expose native review request, list, and show tools in the task toolset.
- MUST add service-level read authority for a single run review.
- MUST reject malformed review payloads before service calls.
</requirements>

## Subtasks
- [x] Add builtin IDs and descriptors for review request/list/show.
- [x] Implement daemon native review bindings.
- [x] Add `task.Manager.GetRunReview` authority.
- [x] Add descriptor, routing, and invalid-payload tests.

## Implementation Details
Required skill activation must match the touched surfaces: backend tasks use `agh-code-guidelines`, `golang-pro`, and `agh-test-conventions`; contract tasks also use `agh-contract-codegen-coship`; web tasks use the web instructions and frontend/design skills; docs tasks use `documentation-writer`, `copywriting` when public prose changes, and the site instructions; QA tasks use the QA skills named in the task. Use the TechSpec and ADRs for architecture; this task records scope and evidence boundaries.

### Relevant Files
- `internal/tools/builtin/tasks.go` - review tool descriptors.
- `internal/daemon/native_review_tools.go` - native bindings.
- `internal/task/manager_review.go` - service read authority.

### Dependent Files
- `internal/daemon/native_tools_test.go` - native review tests.
- `internal/task/manager_review_test.go` - service authority tests.

### Related ADRs
- [ADR-007: Post-Terminal Review Gate](adrs/adr-007.md) - reviews happen after terminal runs and continuations are explicit runs.

## Extensibility / Agent Manageability / Config Lifecycle
- Extensibility: Adds model-facing review management tools.
- Agent manageability: HTTP/UDS/CLI review surfaces are handled in task 19.
- Config lifecycle: No config changes.

### Web/Docs Impact
- `web/`: Review queue/read-only states planned in task 27.
- `packages/site`: Review tool docs planned in task 29.

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
- State: `state.yaml.progress.checklist` iteration 35 is `completed`.
- Memory: `memory/free-iter-034.md`.
- Verification: the workflow memory records final `make verify` PASS for this slice.

## Success Criteria
- All tests passing.
- Test coverage >=80% for changed code paths, or 100% documented evidence coverage for docs-only tasks.
- `make verify` passes before the task is marked complete.
- The task evidence is recorded in workflow memory or QA artifacts.
