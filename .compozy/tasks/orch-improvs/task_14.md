---
status: completed
title: "Bundled Orchestration and Reviewer Skills"
type: backend
complexity: medium
dependencies:
  - task_08
  - task_09
---

# Task 14: Bundled Orchestration and Reviewer Skills

## Overview
This task authored bundled orchestration skills for coordinator, worker, and reviewer sessions. The skills encode deterministic AGH metadata and guardrails so sessions use task context, claims, profiles, and review tools correctly.

<critical>
- ALWAYS READ `_techspec.md`, `_techspec_orchestration.md`, `_techspec_review_gate.md`, every ADR, and the dependency task files before starting.
- REFERENCE TECHSPEC for implementation details; do not duplicate architecture or code snippets here.
- FOCUS ON WHAT needs to be delivered, keep changes scoped, and avoid compatibility shims or fallback paths.
- TESTS REQUIRED: every production change must ship focused tests and run the listed verification gates.
</critical>

<requirements>
- MUST add bundled `agh-orchestrator`, `agh-task-worker`, and `agh-task-reviewer` skills.
- MUST include deterministic `metadata.agh` contracts for loader validation.
- MUST test embed discovery, parsing, metadata, and guardrail content.
</requirements>

## Subtasks
- [x] Author bundled worker skill guidance.
- [x] Author bundled orchestrator/coordinator skill guidance.
- [x] Author bundled reviewer skill guidance requiring review requests.
- [x] Extend bundled loader and registry coverage.

## Implementation Details
Required skill activation must match the touched surfaces: backend tasks use `agh-code-guidelines`, `golang-pro`, and `agh-test-conventions`; contract tasks also use `agh-contract-codegen-coship`; web tasks use the web instructions and frontend/design skills; docs tasks use `documentation-writer`, `copywriting` when public prose changes, and the site instructions; QA tasks use the QA skills named in the task. Use the TechSpec and ADRs for architecture; this task records scope and evidence boundaries.

### Relevant Files
- `internal/skills/bundled/skills/agh-orchestrator/SKILL.md` - coordinator guidance.
- `internal/skills/bundled/skills/agh-task-worker/SKILL.md` - worker guidance.
- `internal/skills/bundled/skills/agh-task-reviewer/SKILL.md` - reviewer guidance.
- `internal/skills/bundled/bundled_test.go` - bundled coverage.

### Dependent Files
- `internal/daemon/native_review_tools.go` - reviewer-bound tool availability depends on reviewer metadata.
- `internal/tools/builtin` - toolset descriptors consumed by skills.

### Related ADRs
- [ADR-007: Post-Terminal Review Gate](adrs/adr-007.md) - reviews happen after terminal runs and continuations are explicit runs.

## Extensibility / Agent Manageability / Config Lifecycle
- Extensibility: Adds bundled capability guidance for AGH task orchestration sessions.
- Agent manageability: Skills instruct agents to use native/transport tools rather than hidden internal state.
- Config lifecycle: No config changes.

### Web/Docs Impact
- `web/`: No direct `web/` change.
- `packages/site`: Bundled skill docs planned in task 29.

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
- State: `state.yaml.progress.checklist` iteration 29 is `completed`.
- Memory: `memory/free-iter-028.md`.
- Verification: the workflow memory records final `make verify` PASS for this slice.

## Success Criteria
- All tests passing.
- Test coverage >=80% for changed code paths, or 100% documented evidence coverage for docs-only tasks.
- `make verify` passes before the task is marked complete.
- The task evidence is recorded in workflow memory or QA artifacts.
