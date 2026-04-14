---
status: completed
title: "Add task and run API contracts plus core handlers"
type: backend
complexity: high
dependencies:
  - task_04
  - task_05
---

# Task 07: Add task and run API contracts plus core handlers

## Overview
Add the shared contract and handler layer that lets daemon transports expose the task domain without duplicating business rules. This task converts the new manager capabilities into validated requests, payloads, and error mappings that both HTTP and UDS can share.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
1. Shared API contracts MUST represent all task and run operations needed by the TechSpec, including task create/list/get/update, child creation, dependency management, and run lifecycle actions.
2. Core handlers MUST validate request inputs, derive transport-safe responses, and map task-domain errors to stable transport statuses without reimplementing business rules.
3. Handler and payload code MUST preserve the identity, ownership, channel, and lifecycle semantics defined in the task domain rather than flattening or renaming them inconsistently.
</requirements>

## Subtasks
- [x] 7.1 Add shared request and response payloads for task and run operations under `internal/api/contract`.
- [x] 7.2 Add core service interfaces and handler methods for task and run operations.
- [x] 7.3 Add request parsing and validation for scope, workspace, owner, channel, and lifecycle operations.
- [x] 7.4 Add error mapping for task-domain validation, authorization, not-found, and invalid-transition failures.
- [x] 7.5 Add payload conversion helpers for task summaries, task details, runs, dependencies, and audit-facing fields.

## Implementation Details
Use the TechSpec sections "API Surface", "Actor and Identity Model", and "Authorization Contract". Follow the patterns already present in `automation.go`, `network.go`, `payloads.go`, and `errors.go` so new task handlers fit the existing API/core organization.

### Relevant Files
- `internal/api/contract/contract.go` — Existing shared contract entrypoint and test pattern.
- `internal/api/contract/responses.go` — Reference response-model style for shared payloads.
- `internal/api/core/interfaces.go` — Service interface surface consumed by handlers.
- `internal/api/core/handlers.go` — Central handler composition point that new task handlers must join.
- `internal/api/core/errors.go` — Shared transport error mapping patterns.
- `internal/api/core/parsers.go` — Existing request parsing patterns for typed surfaces.
- `internal/api/core/payloads.go` — Existing payload builder location for shared responses.

### Dependent Files
- `internal/api/httpapi/routes.go` — Will map HTTP routes onto the core handlers added here.
- `internal/api/udsapi/routes.go` — Will map UDS routes onto the same core handlers.
- `internal/cli/task.go` — Will depend on the new contract payloads and endpoints.

### Related ADRs
- [ADR-003: Use Queue-First TaskRun Lifecycle with Central TaskManager Authority](../adrs/adr-003.md) — Governs which transitions the API may expose.
- [ADR-004: Support Optional Task-to-Network-Channel Binding](../adrs/adr-004.md) — Governs channel fields that must be preserved in payloads.
- [ADR-005: Derive Actor Identity Server-Side and Allow Optional Mutable Ownership](../adrs/adr-005.md) — Governs identity and owner fields that handlers must treat correctly.

## Deliverables
- Shared task/run request and response contracts.
- Core task/run handlers, parsers, payload builders, and error mappings.
- Handler tests covering validation and status mapping behavior.
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration tests for handler-to-manager flows **(REQUIRED)**

## Tests
- Unit tests:
  - [x] Verify invalid scope, workspace, owner, and channel inputs are rejected with stable validation errors.
  - [x] Verify task-domain not-found, invalid-transition, and authorization errors map to the correct transport statuses.
  - [x] Verify payload builders preserve immutable `created_by`, immutable `origin`, optional `owner`, and run/session attachment fields.
- Integration tests:
  - [x] Verify a create-task request reaches the manager with the expected parsed filters and actor context envelope.
  - [x] Verify run lifecycle handler calls sequence correctly against a fake manager implementation with no duplicate business logic in the handler layer.
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- HTTP and UDS transports can share one validated task/run contract and core handler layer
- API/core remains a thin orchestration layer over the task domain rather than a second rule engine
