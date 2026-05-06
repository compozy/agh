---
status: completed
title: "Reviewer-Bound Native `submit_run_review` Tool"
type: backend
complexity: high
dependencies:
  - task_09
  - task_10
  - task_14
---

# Task 15: Reviewer-Bound Native `submit_run_review` Tool

## Overview
This task added the reviewer-bound native `submit_run_review` tool. The tool is hidden without an active review request binding, validates review/run identity, normalizes verdict payloads, and delegates all authority to `task.Service.RecordRunReview`.

<critical>
- ALWAYS READ `_techspec.md`, `_techspec_orchestration.md`, `_techspec_review_gate.md`, every ADR, and the dependency task files before starting.
- REFERENCE TECHSPEC for implementation details; do not duplicate architecture or code snippets here.
- FOCUS ON WHAT needs to be delivered, keep changes scoped, and avoid compatibility shims or fallback paths.
- TESTS REQUIRED: every production change must ship focused tests and run the listed verification gates.
</critical>

<requirements>
- MUST expose `submit_run_review` only to eligible reviewer sessions.
- MUST validate the active review binding before any verdict write.
- MUST delegate verdict recording to task-service authority.
</requirements>

## Subtasks
- [x] Add builtin tool ID and descriptor.
- [x] Register the model-facing tool in the autonomy toolset.
- [x] Implement daemon native binding with reviewer-session gating.
- [x] Add native tool tests for success, hidden, and invalid input paths.

## Implementation Details
Required skill activation must match the touched surfaces: backend tasks use `agh-code-guidelines`, `golang-pro`, and `agh-test-conventions`; contract tasks also use `agh-contract-codegen-coship`; web tasks use the web instructions and frontend/design skills; docs tasks use `documentation-writer`, `copywriting` when public prose changes, and the site instructions; QA tasks use the QA skills named in the task. Use the TechSpec and ADRs for architecture; this task records scope and evidence boundaries.

### Relevant Files
- `internal/tools/builtin_ids.go` - builtin IDs.
- `internal/tools/builtin/autonomy.go` - descriptor.
- `internal/daemon/native_review_tools.go` - native binding.
- `internal/daemon/native_tools.go` - registry wiring.

### Dependent Files
- `internal/tools/builtin/builtin_test.go` - descriptor coverage.
- `internal/daemon/native_tools_test.go` - native routing tests.

### Related ADRs
- [ADR-007: Post-Terminal Review Gate](adrs/adr-007.md) - reviews happen after terminal runs and continuations are explicit runs.

## Extensibility / Agent Manageability / Config Lifecycle
- Extensibility: Model-facing reviewer tool extends runtime autonomy without exposing claim tokens.
- Agent manageability: HTTP/UDS/CLI verdict submission is handled in task 19.
- Config lifecycle: No config changes.

### Web/Docs Impact
- `web/`: Review verdict UI planned in task 27.
- `packages/site`: Tool behavior documented in task 29.

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
- State: `state.yaml.progress.checklist` iteration 31 is `completed`.
- Memory: `memory/free-iter-030.md`.
- Verification: the workflow memory records final `make verify` PASS for this slice.

## Success Criteria
- All tests passing.
- Test coverage >=80% for changed code paths, or 100% documented evidence coverage for docs-only tasks.
- `make verify` passes before the task is marked complete.
- The task evidence is recorded in workflow memory or QA artifacts.
