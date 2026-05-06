---
status: completed
title: "Profile-Based Claim Eligibility Filtering"
type: backend
complexity: high
dependencies:
  - task_08
---

# Task 11: Profile-Based Claim Eligibility Filtering

## Overview
This task applied execution-profile worker and participant eligibility filters to `ClaimNextRun`. It blocks claims from wrong agents or claimers missing profile-required capabilities without mutating queued runs.

<critical>
- ALWAYS READ `_techspec.md`, `_techspec_orchestration.md`, `_techspec_review_gate.md`, every ADR, and the dependency task files before starting.
- REFERENCE TECHSPEC for implementation details; do not duplicate architecture or code snippets here.
- FOCUS ON WHAT needs to be delivered, keep changes scoped, and avoid compatibility shims or fallback paths.
- TESTS REQUIRED: every production change must ship focused tests and run the listed verification gates.
</critical>

<requirements>
- MUST enforce profile worker and participant agent eligibility in claim selection.
- MUST enforce worker and participant required capabilities during claims.
- MUST leave queued runs unchanged when claim criteria are ineligible.
</requirements>

## Subtasks
- [x] Extend `ClaimNextRun` filters for worker agent selectors.
- [x] Extend capability matching for profile-required capabilities.
- [x] Preserve queued state for rejected claim attempts.
- [x] Add positive and negative claim eligibility tests.

## Implementation Details
Required skill activation must match the touched surfaces: backend tasks use `agh-code-guidelines`, `golang-pro`, and `agh-test-conventions`; contract tasks also use `agh-contract-codegen-coship`; web tasks use the web instructions and frontend/design skills; docs tasks use `documentation-writer`, `copywriting` when public prose changes, and the site instructions; QA tasks use the QA skills named in the task. Use the TechSpec and ADRs for architecture; this task records scope and evidence boundaries.

### Relevant Files
- `internal/store/globaldb/global_db_task_claim.go` - claim filtering.
- `internal/store/globaldb/global_db_task_claim_test.go` - eligibility coverage.

### Dependent Files
- `internal/task/profile.go` - profile selector semantics.
- `internal/store/globaldb/global_db_task_profile.go` - selector persistence.

### Related ADRs
- [ADR-010: Typed Overlay](adrs/adr-010.md) - typed orchestration/profile overlays instead of metadata JSON.

## Extensibility / Agent Manageability / Config Lifecycle
- Extensibility: Profiles directly control worker eligibility at runtime.
- Agent manageability: Agents can configure these filters through tasks 16 and 18 surfaces.
- Config lifecycle: Uses execution profile selector state; no new config keys.

### Web/Docs Impact
- `web/`: Profile editor must make eligibility visible in task 27.
- `packages/site`: Eligibility semantics documented in task 28.

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
- State: `state.yaml.progress.checklist` iteration 23 is `completed`.
- Memory: `memory/free-iter-022.md`.
- Verification: the workflow memory records final `make verify` PASS for this slice.

## Success Criteria
- All tests passing.
- Test coverage >=80% for changed code paths, or 100% documented evidence coverage for docs-only tasks.
- `make verify` passes before the task is marked complete.
- The task evidence is recorded in workflow memory or QA artifacts.
