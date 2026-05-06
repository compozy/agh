---
status: completed
title: "Execution Profile HTTP, UDS, CLI, and OpenAPI Surfaces"
type: backend
complexity: high
dependencies:
  - task_16
---

# Task 18: Execution Profile HTTP, UDS, CLI, and OpenAPI Surfaces

## Overview
This task exposed task execution profiles through HTTP, UDS, CLI, and OpenAPI. It added shared handlers, route mounts, CLI verbs, generated OpenAPI and TypeScript contracts, and CLI reference docs.

<critical>
- ALWAYS READ `_techspec.md`, `_techspec_orchestration.md`, `_techspec_review_gate.md`, every ADR, and the dependency task files before starting.
- REFERENCE TECHSPEC for implementation details; do not duplicate architecture or code snippets here.
- FOCUS ON WHAT needs to be delivered, keep changes scoped, and avoid compatibility shims or fallback paths.
- TESTS REQUIRED: every production change must ship focused tests and run the listed verification gates.
</critical>

<requirements>
- MUST expose inspect, update, and delete profile operations across HTTP, UDS, and CLI.
- MUST generate OpenAPI and TypeScript contracts in the same change.
- MUST regenerate CLI reference docs for new verbs.
</requirements>

## Subtasks
- [x] Add contract DTOs and shared API core handlers.
- [x] Mount HTTP and UDS routes for profile operations.
- [x] Add CLI client methods and `agh task profile` commands.
- [x] Run codegen, codegen-check, CLI docs, site build, and tests.

## Implementation Details
Required skill activation must match the touched surfaces: backend tasks use `agh-code-guidelines`, `golang-pro`, and `agh-test-conventions`; contract tasks also use `agh-contract-codegen-coship`; web tasks use the web instructions and frontend/design skills; docs tasks use `documentation-writer`, `copywriting` when public prose changes, and the site instructions; QA tasks use the QA skills named in the task. Use the TechSpec and ADRs for architecture; this task records scope and evidence boundaries.

### Relevant Files
- `internal/api/contract/tasks.go` - profile DTOs.
- `internal/api/core/tasks.go` - shared handlers.
- `internal/api/httpapi/routes.go` - HTTP routes.
- `internal/api/udsapi/routes.go` - UDS routes.
- `internal/cli/task.go` - CLI commands.

### Dependent Files
- `internal/api/spec/spec.go` - OpenAPI operations.
- `openapi/agh.json` - generated OpenAPI.
- `web/src/generated/agh-openapi.d.ts` - generated TS types.
- `packages/site/content/runtime/cli-reference/task/profile/` - CLI docs.

### Related ADRs
- [ADR-010: Typed Overlay](adrs/adr-010.md) - typed orchestration/profile overlays instead of metadata JSON.

## Extensibility / Agent Manageability / Config Lifecycle
- Extensibility: Profile authority becomes agent-operable through public transports.
- Agent manageability: Agents can manage profiles via CLI, HTTP, UDS, and native tools.
- Config lifecycle: No new config keys; surfaces manage persisted per-task profile state.

### Web/Docs Impact
- `web/`: Generated TS types are available for tasks 26 and 27.
- `packages/site`: Generated CLI docs exist; narrative docs deferred to task 28.

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
- State: `state.yaml.progress.checklist` iteration 37 is `completed`.
- Memory: `memory/free-iter-036.md`.
- Verification: the workflow memory records final `make verify` PASS for this slice.

## Success Criteria
- All tests passing.
- Test coverage >=80% for changed code paths, or 100% documented evidence coverage for docs-only tasks.
- `make verify` passes before the task is marked complete.
- The task evidence is recorded in workflow memory or QA artifacts.
