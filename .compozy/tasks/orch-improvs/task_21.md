---
status: completed
title: "Bridge Notification Transport Consolidation and Spec Alignment"
type: backend
complexity: high
dependencies:
  - task_07
  - task_18
  - task_20
---

# Task 21: Bridge Notification Transport Consolidation and Spec Alignment

## Overview
This task closes the current bridge notification transport slice and aligns its contract with the approved TechSpec. The code path has been partially implemented, but it must be reconciled with the required `/api/tasks/{id}/notifications/bridges` route shape, contract codegen, CLI docs, and closure memory/state before it is marked complete.

<critical>
- ALWAYS READ `_techspec.md`, `_techspec_orchestration.md`, `_techspec_review_gate.md`, every ADR, and the dependency task files before starting.
- REFERENCE TECHSPEC for implementation details; do not duplicate architecture or code snippets here.
- FOCUS ON WHAT needs to be delivered, keep changes scoped, and avoid compatibility shims or fallback paths.
- TESTS REQUIRED: every production change must ship focused tests and run the listed verification gates.
</critical>

<requirements>
- MUST expose bridge terminal notification subscription create, list, show, delete, and diagnostics across HTTP, UDS, CLI, and OpenAPI.
- MUST use the TechSpec route shape `/api/tasks/{id}/notifications/bridges` instead of divergent bridge-notification route names.
- MUST co-ship OpenAPI, TypeScript contracts, CLI docs, and route coverage tests.
</requirements>

## Subtasks
- [x] Finish shared contract DTOs and API core handlers for bridge notification subscriptions.
- [x] Align HTTP, UDS, OpenAPI, and CLI paths/names with the TechSpec notification route contract.
- [x] Add focused tests for route binding, structured CLI output, deterministic errors, and generated contracts.
- [x] Run `make codegen`, `make codegen-check`, `make cli-docs`, and `make verify`.
- [x] Close the current cy-codex-loop slice with memory and managed state updates.

## Implementation Details
Required skill activation must match the touched surfaces: backend tasks use `agh-code-guidelines`, `golang-pro`, and `agh-test-conventions`; contract tasks also use `agh-contract-codegen-coship`; web tasks use the web instructions and frontend/design skills; docs tasks use `documentation-writer`, `copywriting` when public prose changes, and the site instructions; QA tasks use the QA skills named in the task. Use the TechSpec and ADRs for architecture; this task records scope and evidence boundaries.

### Relevant Files
- `internal/api/contract/tasks.go` - notification subscription DTOs.
- `internal/api/core` - shared task notification handlers.
- `internal/api/httpapi/routes.go` - HTTP route alignment.
- `internal/api/udsapi/routes.go` - UDS route alignment.
- `internal/cli/task.go` - CLI notification commands.

### Dependent Files
- `internal/api/spec/spec.go` - OpenAPI operations.
- `openapi/agh.json` - generated OpenAPI.
- `web/src/generated/agh-openapi.d.ts` - generated TS types.
- `packages/site/content/runtime/cli-reference/task/` - generated CLI docs.

### Related ADRs
- [ADR-003: Durable Cursor Primitive](adrs/adr-003.md) - notification cursors are monotonic and replay-safe.

## Extensibility / Agent Manageability / Config Lifecycle
- Extensibility: Bridge notification subscriptions become publicly manageable runtime primitives.
- Agent manageability: Agents must be able to create/list/show/delete subscriptions through CLI, HTTP, UDS, and native bridge paths.
- Config lifecycle: No config keys expected; cursor identity and bridge target are persisted runtime state.

### Web/Docs Impact
- `web/`: Generated notification types feed tasks 26 and 27.
- `packages/site`: Generated CLI reference is required here; narrative docs are task 29.

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
- State: completed through managed `state.yaml` task reconciliation and `task_21` closure.
- Implementation note: canonical `/api/tasks/{id}/notifications/bridges` route alignment, generated contract/doc co-ship, and task closure memory are complete. Full cursor diagnostic expansion remains task_25.

## Success Criteria
- All tests passing.
- Test coverage >=80% for changed code paths, or 100% documented evidence coverage for docs-only tasks.
- `make verify` passes before the task is marked complete.
- The task evidence is recorded in workflow memory or QA artifacts.
