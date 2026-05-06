---
status: completed
title: "Fresh/Migrated Schema Drift Guard for Execution Profiles"
type: backend
complexity: medium
dependencies:
  - task_02
---

# Task 03: Fresh/Migrated Schema Drift Guard for Execution Profiles

## Overview
This task removed drift between fresh GlobalDB creation and migration-applied databases for execution profiles. It consolidated DDL primitives and added guard tests so future schema edits cannot silently diverge.

<critical>
- ALWAYS READ `_techspec.md`, `_techspec_orchestration.md`, `_techspec_review_gate.md`, every ADR, and the dependency task files before starting.
- REFERENCE TECHSPEC for implementation details; do not duplicate architecture or code snippets here.
- FOCUS ON WHAT needs to be delivered, keep changes scoped, and avoid compatibility shims or fallback paths.
- TESTS REQUIRED: every production change must ship focused tests and run the listed verification gates.
</critical>

<requirements>
- MUST share DDL between fresh schema creation and migrations.
- MUST remove positional index dependencies around current-run DDL.
- MUST fail tests when fresh and migrated schemas diverge.
</requirements>

## Subtasks
- [x] Extract shared orchestration profile schema statements.
- [x] Use shared statements in fresh GlobalDB creation and migration v17.
- [x] Add explicit current-run index statement dependency.
- [x] Add drift guard tests for fresh and migrated databases.

## Implementation Details
Required skill activation must match the touched surfaces: backend tasks use `agh-code-guidelines`, `golang-pro`, and `agh-test-conventions`; contract tasks also use `agh-contract-codegen-coship`; web tasks use the web instructions and frontend/design skills; docs tasks use `documentation-writer`, `copywriting` when public prose changes, and the site instructions; QA tasks use the QA skills named in the task. Use the TechSpec and ADRs for architecture; this task records scope and evidence boundaries.

### Relevant Files
- `internal/store/globaldb/schema_task_orchestration_profile.go` - shared DDL source.
- `internal/store/globaldb/schema.go` - fresh schema assembly.
- `internal/store/globaldb/migrate_task_orchestration_profile.go` - migration reuse of shared DDL.

### Dependent Files
- `internal/store/globaldb/schema_test.go` - exact schema guard.
- `internal/store/globaldb/migrations_test.go` - migration guard.

### Related ADRs
- [ADR-010: Typed Overlay](adrs/adr-010.md) - typed orchestration/profile overlays instead of metadata JSON.

## Extensibility / Agent Manageability / Config Lifecycle
- Extensibility: Hardened profile storage as a reusable runtime primitive.
- Agent manageability: No user-facing surface; protects later agent-operable profile APIs.
- Config lifecycle: No config changes.

### Web/Docs Impact
- `web/`: No direct `web/` change.
- `packages/site`: No direct `packages/site` change.

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
- State: `state.yaml.progress.checklist` iteration 7 is `completed`.
- Memory: `memory/free-iter-006.md`.
- Verification: the workflow memory records final `make verify` PASS for this slice.

## Success Criteria
- All tests passing.
- Test coverage >=80% for changed code paths, or 100% documented evidence coverage for docs-only tasks.
- `make verify` passes before the task is marked complete.
- The task evidence is recorded in workflow memory or QA artifacts.
