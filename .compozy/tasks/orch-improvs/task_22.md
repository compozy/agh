---
status: completed
title: "ReviewRouter Runtime Routing and Reviewer Binding Orchestration"
type: backend
complexity: critical
dependencies:
  - task_10
  - task_13
  - task_14
  - task_15
  - task_17
  - task_19
---

# Task 22: ReviewRouter Runtime Routing and Reviewer Binding Orchestration

## Overview
This task implements the missing daemon-side review routing path. It must wake the coordinator/review router after review requests, select reviewer sessions using review profile selectors, exclude the original worker when required, bind reviewer sessions, and fix the review-store typed-error hygiene found during branch audit.

<critical>
- ALWAYS READ `_techspec.md`, `_techspec_orchestration.md`, `_techspec_review_gate.md`, every ADR, and the dependency task files before starting.
- REFERENCE TECHSPEC for implementation details; do not duplicate architecture or code snippets here.
- FOCUS ON WHAT needs to be delivered, keep changes scoped, and avoid compatibility shims or fallback paths.
- TESTS REQUIRED: every production change must ship focused tests and run the listed verification gates.
</critical>

<requirements>
- MUST add a ReviewRouter wake/callback path from review request creation to reviewer-session routing.
- MUST enforce review profile peer, channel, capability, and original-worker exclusion rules before binding.
- MUST replace string-based SQLite unique-constraint detection in review persistence with typed `errors.Is`/`errors.As` handling.
</requirements>

## Subtasks
- [x] Add review-request wake/callback integration in the daemon/coordinator composition root.
- [x] Implement reviewer route selection and binding using task review profile selectors and active session capabilities.
- [x] Add original-worker exclusion and deterministic no-reviewer diagnostics.
- [x] Replace `strings.Contains(err.Error(), "uq_")` review-store checks with typed sqlite constraint handling.
- [x] Add integration tests for request -> route -> bind and typed replay/constraint behavior.

## Implementation Details
Required skill activation must match the touched surfaces: backend tasks use `agh-code-guidelines`, `golang-pro`, and `agh-test-conventions`; contract tasks also use `agh-contract-codegen-coship`; web tasks use the web instructions and frontend/design skills; docs tasks use `documentation-writer`, `copywriting` when public prose changes, and the site instructions; QA tasks use the QA skills named in the task. Use the TechSpec and ADRs for architecture; this task records scope and evidence boundaries.

### Relevant Files
- `internal/task/manager_review.go` - review request authority.
- `internal/store/globaldb/global_db_task_review.go` - review persistence and constraint hygiene.
- `internal/daemon` - coordinator/runtime wiring.
- `internal/coordinator` - reviewer routing composition point if present.

### Dependent Files
- `internal/skills/bundled/skills/agh-task-reviewer/SKILL.md` - reviewer capability expectations.
- `internal/daemon/native_review_tools.go` - native review tools consumed by reviewers.
- `internal/api/core/task_reviews.go` - transport request entry point.

### Related ADRs
- [ADR-007: Post-Terminal Review Gate](adrs/adr-007.md) - reviews happen after terminal runs and continuations are explicit runs.

## Extensibility / Agent Manageability / Config Lifecycle
- Extensibility: Connects review policy to runtime reviewer-session orchestration.
- Agent manageability: Agents request reviews through native/CLI/HTTP/UDS surfaces and receive deterministic routing or no-route diagnostics.
- Config lifecycle: Consumes review profile policy and existing task orchestration config; no fallback defaults beyond typed config.

### Web/Docs Impact
- `web/`: Review routing state and diagnostics must be queryable by tasks 26 and 27.
- `packages/site`: Review routing behavior documented in task 29.

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
- State: completed through managed `state.yaml` task closure after `make verify` passed.
- Implementation note: review request creation now notifies a daemon-side review router, the router selects or creates eligible reviewer sessions using persisted review profile selectors, original worker sessions are excluded before binding, deterministic no-route diagnostics are recorded through task service authority, and review-store UNIQUE conflicts are classified through typed SQLite errors instead of string matching.
- Verification: focused task/daemon/globaldb tests, focused race tests, `make lint`, and final `make verify` passed. The first full gate hit an unrelated transient TypeScript SDK integration timeout; the focused SDK integration test passed immediately before the final full gate succeeded.

## Success Criteria
- All tests passing.
- Test coverage >=80% for changed code paths, or 100% documented evidence coverage for docs-only tasks.
- `make verify` passes before the task is marked complete.
- The task evidence is recorded in workflow memory or QA artifacts.
