---
status: completed
title: "Formal Remaining-Work Cross-Walk and QA Tail"
type: docs
complexity: medium
dependencies:
  - task_19
---

# Task 20: Formal Remaining-Work Cross-Walk and QA Tail

## Overview
This task created an initial remaining-work cross-walk after backend slices had landed. It generated `_tasks.md` plus task files for the remaining implementation, web/docs, lessons, QA, and post-QA review gates, then validated the task package.

<critical>
- ALWAYS READ `_techspec.md`, `_techspec_orchestration.md`, `_techspec_review_gate.md`, every ADR, and the dependency task files before starting.
- REFERENCE TECHSPEC for implementation details; do not duplicate architecture or code snippets here.
- FOCUS ON WHAT needs to be delivered, keep changes scoped, and avoid compatibility shims or fallback paths.
- TESTS REQUIRED: every production change must ship focused tests and run the listed verification gates.
</critical>

<requirements>
- MUST document remaining work instead of relying on free-mode memory alone.
- MUST include QA report and QA execution tail tasks.
- MUST validate the task package with `compozy tasks validate`.
</requirements>

## Subtasks
- [x] Create an initial `_tasks.md` for remaining work.
- [x] Create task files for backend remainder, web, site docs, lessons, QA report, and QA execution.
- [x] Use schema-valid task metadata and QA task types.
- [x] Validate the task package and run the full verify gate.

## Implementation Details
Required skill activation must match the touched surfaces: backend tasks use `agh-code-guidelines`, `golang-pro`, and `agh-test-conventions`; contract tasks also use `agh-contract-codegen-coship`; web tasks use the web instructions and frontend/design skills; docs tasks use `documentation-writer`, `copywriting` when public prose changes, and the site instructions; QA tasks use the QA skills named in the task. Use the TechSpec and ADRs for architecture; this task records scope and evidence boundaries.

### Relevant Files
- `.compozy/tasks/orch-improvs/_tasks.md` - initial remaining task list.
- `.compozy/tasks/orch-improvs/task_01.md` through `task_06.md` - initial remaining task files.
- `.agents/skills/cy-create-tasks` - task-generation rules.

### Dependent Files
- `.compozy/tasks/orch-improvs/state.yaml` - free-mode loop state.
- `.compozy/tasks/orch-improvs/memory/MEMORY.md` - workflow memory.

### Related ADRs
- [ADR-003: Durable Cursor Primitive](adrs/adr-003.md) - notification cursor and replay semantics.
- [ADR-005: Denormalized Current Run Projection](adrs/adr-005.md) - current run projection boundaries.
- [ADR-007: Post-Terminal Review Gate](adrs/adr-007.md) - review request/verdict/continuation authority.
- [ADR-010: Typed Overlay](adrs/adr-010.md) - execution profile schema and config overlay shape.

## Extensibility / Agent Manageability / Config Lifecycle
- Extensibility: Makes remaining extensibility/manageability gaps explicit in task files.
- Agent manageability: Frames remaining agent-operable surfaces as executable work.
- Config lifecycle: No config changes.

### Web/Docs Impact
- `web/`: The first pass identified web work but did not include historical completed tasks.
- `packages/site`: The first pass identified site and memory lesson work but did not include full tracking.

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
- State: `state.yaml.progress.checklist` iteration 41 is `completed`.
- Memory: `memory/free-iter-040.md`.
- Verification: the workflow memory records final `make verify` PASS for this slice.

## Success Criteria
- All tests passing.
- Test coverage >=80% for changed code paths, or 100% documented evidence coverage for docs-only tasks.
- `make verify` passes before the task is marked complete.
- The task evidence is recorded in workflow memory or QA artifacts.
