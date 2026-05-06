---
status: completed
title: "Bridge Task Subscription Store and Terminal Notifier"
type: backend
complexity: high
dependencies:
  - task_06
---

# Task 07: Bridge Task Subscription Store and Terminal Notifier

## Overview
This task added bridge task subscription persistence and a terminal notifier consumer over durable task events. It ensured bridge delivery uses cursor state and records delivery errors without advancing failed cursors.

<critical>
- ALWAYS READ `_techspec.md`, `_techspec_orchestration.md`, `_techspec_review_gate.md`, every ADR, and the dependency task files before starting.
- REFERENCE TECHSPEC for implementation details; do not duplicate architecture or code snippets here.
- FOCUS ON WHAT needs to be delivered, keep changes scoped, and avoid compatibility shims or fallback paths.
- TESTS REQUIRED: every production change must ship focused tests and run the listed verification gates.
</critical>

<requirements>
- MUST persist bridge terminal notification subscriptions with stable cursor identity.
- MUST deliver terminal task events through durable cursor replay.
- MUST record delivery failure diagnostics without advancing the cursor.
</requirements>

## Subtasks
- [x] Add `bridge_task_subscriptions` schema and migration.
- [x] Implement subscription CRUD/list/delete in GlobalDB.
- [x] Add terminal task notifier replay and delivery logic.
- [x] Add tests for delivery, failure, superseded event skipping, and cursor error recording.

## Implementation Details
Required skill activation must match the touched surfaces: backend tasks use `agh-code-guidelines`, `golang-pro`, and `agh-test-conventions`; contract tasks also use `agh-contract-codegen-coship`; web tasks use the web instructions and frontend/design skills; docs tasks use `documentation-writer`, `copywriting` when public prose changes, and the site instructions; QA tasks use the QA skills named in the task. Use the TechSpec and ADRs for architecture; this task records scope and evidence boundaries.

### Relevant Files
- `internal/bridges` - bridge subscription and terminal notifier logic.
- `internal/store/globaldb/global_db_bridge_task_subscription.go` - subscription store.
- `internal/store/globaldb/global_db_notification_cursor.go` - cursor diagnostics.

### Dependent Files
- `internal/bridges` tests - notifier behavior.
- `internal/store/globaldb` tests - subscription and cursor persistence.

### Related ADRs
- [ADR-003: Durable Cursor Primitive](adrs/adr-003.md) - notification cursors are monotonic and replay-safe.

## Extensibility / Agent Manageability / Config Lifecycle
- Extensibility: Bridge SDK/runtime gains a durable terminal notification subscription primitive.
- Agent manageability: Transport manageability is completed in tasks 21 and 25.
- Config lifecycle: No config changes.

### Web/Docs Impact
- `web/`: Notification diagnostics UI planned in tasks 26 and 27.
- `packages/site`: Bridge notification docs planned in task 29.

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
- State: `state.yaml.progress.checklist` iteration 15 is `completed`.
- Memory: `memory/free-iter-014.md`.
- Verification: the workflow memory records final `make verify` PASS for this slice.

## Success Criteria
- All tests passing.
- Test coverage >=80% for changed code paths, or 100% documented evidence coverage for docs-only tasks.
- `make verify` passes before the task is marked complete.
- The task evidence is recorded in workflow memory or QA artifacts.
