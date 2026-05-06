---
status: completed
title: "Run Review HTTP, UDS, CLI, and OpenAPI Surfaces"
type: backend
complexity: high
dependencies:
  - task_15
  - task_17
---

# Task 19: Run Review HTTP, UDS, CLI, and OpenAPI Surfaces

## Overview
This task exposed run review request, list, show, and verdict submission operations across HTTP, UDS, CLI, and OpenAPI. It kept authority in `task.Service` and generated contract and CLI reference artifacts.

<critical>
- ALWAYS READ `_techspec.md`, `_techspec_orchestration.md`, `_techspec_review_gate.md`, every ADR, and the dependency task files before starting.
- REFERENCE TECHSPEC for implementation details; do not duplicate architecture or code snippets here.
- FOCUS ON WHAT needs to be delivered, keep changes scoped, and avoid compatibility shims or fallback paths.
- TESTS REQUIRED: every production change must ship focused tests and run the listed verification gates.
</critical>

<requirements>
- MUST expose review request/list/show/verdict submission across HTTP, UDS, and CLI.
- MUST model terminal verdicts as `status=recorded` plus typed `outcome`.
- MUST co-ship OpenAPI, TypeScript contracts, and CLI reference docs.
</requirements>

## Subtasks
- [x] Add review contract DTOs and response wrappers.
- [x] Add shared API handlers for run and review routes.
- [x] Mount HTTP and UDS routes and add CLI commands.
- [x] Run codegen, codegen-check, CLI docs, and verification tests.

## Implementation Details
Required skill activation must match the touched surfaces: backend tasks use `agh-code-guidelines`, `golang-pro`, and `agh-test-conventions`; contract tasks also use `agh-contract-codegen-coship`; web tasks use the web instructions and frontend/design skills; docs tasks use `documentation-writer`, `copywriting` when public prose changes, and the site instructions; QA tasks use the QA skills named in the task. Use the TechSpec and ADRs for architecture; this task records scope and evidence boundaries.

### Relevant Files
- `internal/api/core/task_reviews.go` - shared review handlers.
- `internal/api/contract/tasks.go` - review DTOs.
- `internal/api/httpapi/routes.go` - HTTP routes.
- `internal/api/udsapi/routes.go` - UDS routes.
- `internal/cli/task.go` - review commands.

### Dependent Files
- `internal/api/spec/spec.go` - OpenAPI review operations.
- `openapi/agh.json` - generated OpenAPI.
- `web/src/generated/agh-openapi.d.ts` - generated TS types.
- `packages/site/content/runtime/cli-reference/task/review/` - CLI docs.

### Related ADRs
- [ADR-007: Post-Terminal Review Gate](adrs/adr-007.md) - reviews happen after terminal runs and continuations are explicit runs.

## Extensibility / Agent Manageability / Config Lifecycle
- Extensibility: Review authority becomes agent-operable through public transports.
- Agent manageability: Agents can request, inspect, list, and submit reviews via CLI, HTTP, UDS, and native tools.
- Config lifecycle: No new config keys; review policy/profile state comes from earlier tasks.

### Web/Docs Impact
- `web/`: Generated review types are available for tasks 26 and 27.
- `packages/site`: Generated CLI docs exist; narrative docs deferred to task 29.

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
- State: `state.yaml.progress.checklist` iteration 39 is `completed`.
- Memory: `memory/free-iter-038.md`.
- Verification: the workflow memory records final `make verify` PASS for this slice.

## Success Criteria
- All tests passing.
- Test coverage >=80% for changed code paths, or 100% documented evidence coverage for docs-only tasks.
- `make verify` passes before the task is marked complete.
- The task evidence is recorded in workflow memory or QA artifacts.
