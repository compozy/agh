---
status: completed
title: "Task Orchestration GlobalDB Schema Foundation"
type: backend
complexity: high
dependencies:
  - task_01
---

# Task 02: Task Orchestration GlobalDB Schema Foundation

## Overview
This task created the first persistent schema layer for task orchestration profiles. It added numbered migration coverage for projection fields, run summaries, provenance, and execution-profile selector tables.

<critical>
- ALWAYS READ `_techspec.md`, `_techspec_orchestration.md`, `_techspec_review_gate.md`, every ADR, and the dependency task files before starting.
- REFERENCE TECHSPEC for implementation details; do not duplicate architecture or code snippets here.
- FOCUS ON WHAT needs to be delivered, keep changes scoped, and avoid compatibility shims or fallback paths.
- TESTS REQUIRED: every production change must ship focused tests and run the listed verification gates.
</critical>

<requirements>
- MUST add numbered GlobalDB migration coverage for orchestration profile persistence.
- MUST create selector tables without `metadata_json` escape hatches.
- MUST preserve fresh database and migrated database behavior.
</requirements>

## Subtasks
- [x] Add migration registry entry for orchestration/profile schema.
- [x] Add fresh schema statements for profile selector tables.
- [x] Persist task projection and run-summary fields needed by later slices.
- [x] Add migration and fresh-schema tests.

## Implementation Details
Required skill activation must match the touched surfaces: backend tasks use `agh-code-guidelines`, `golang-pro`, and `agh-test-conventions`; contract tasks also use `agh-contract-codegen-coship`; web tasks use the web instructions and frontend/design skills; docs tasks use `documentation-writer`, `copywriting` when public prose changes, and the site instructions; QA tasks use the QA skills named in the task. Use the TechSpec and ADRs for architecture; this task records scope and evidence boundaries.

### Relevant Files
- `internal/store/globaldb/migrate_task_orchestration_profile.go` - numbered schema migration.
- `internal/store/globaldb/schema_task_orchestration_profile.go` - profile schema DDL.
- `internal/store/globaldb/global_db.go` - migration registry integration.

### Dependent Files
- `internal/store/globaldb/schema_test.go` - schema assertions.
- `internal/store/globaldb/migrations_test.go` - migration behavior.

### Related ADRs
- [ADR-010: Typed Overlay](adrs/adr-010.md) - typed orchestration/profile overlays instead of metadata JSON.

## Extensibility / Agent Manageability / Config Lifecycle
- Extensibility: Created typed storage primitives for extensible execution profiles.
- Agent manageability: No direct CLI/HTTP/UDS surface yet; persistence is consumed by later service and transport tasks.
- Config lifecycle: No additional config keys beyond task 01.

### Web/Docs Impact
- `web/`: No direct `web/` change; generated contracts arrive in task 18.
- `packages/site`: Narrative docs deferred to task 28.

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
- State: `state.yaml.progress.checklist` iteration 5 is `completed`.
- Memory: `memory/free-iter-004.md`.
- Verification: the workflow memory records final `make verify` PASS for this slice.

## Success Criteria
- All tests passing.
- Test coverage >=80% for changed code paths, or 100% documented evidence coverage for docs-only tasks.
- `make verify` passes before the task is marked complete.
- The task evidence is recorded in workflow memory or QA artifacts.
