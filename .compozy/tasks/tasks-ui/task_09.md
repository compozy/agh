---
status: completed
title: HTTP task transport and route wiring
type: backend
complexity: medium
dependencies:
  - task_08
---

# Task 09: HTTP task transport and route wiring

## Overview

Expose the new task surface over the daemon’s HTTP API and keep it aligned with the shared handler layer. This task is complete when the documented routes exist under `/api`, return the shared payloads, and are covered by HTTP integration and parity tests.

<critical>
- ALWAYS READ `_techspec.md`, `task_07.md`, and `task_08.md` before registering routes
- REFERENCE TECHSPEC sections "API Endpoints", "Response and status conventions", and "Development Sequencing"
- FOCUS ON "WHAT" — register and verify the HTTP routes for the shared task surface, not new handler behavior
- MINIMIZE CODE — route registration should stay thin and rely entirely on `api/core`
- TESTS REQUIRED — HTTP integration and parity coverage must prove the new routes actually work
- GREENFIELD: nao registre rotas parcialmente ou fora do contrato; tudo que entrar em HTTP precisa existir no spec e nos handlers compartilhados
</critical>

<requirements>
- MUST register the new task point-read, task-live, and task-observe routes in the HTTP transport
- MUST expose task-native stream routes with the expected path shape and media type
- MUST keep the HTTP route surface aligned with the OpenAPI definitions added in task_07
- MUST add or extend HTTP integration tests for the new task routes
- SHOULD keep route registration grouped coherently under `/api/tasks`, `/api/task-runs`, and `/api/observe/tasks`
</requirements>

## Subtasks
- [x] 9.1 Register the new HTTP task, task-run, and observe-task routes
- [x] 9.2 Verify HTTP route grouping and handler binding stay aligned with the shared task surface
- [x] 9.3 Extend HTTP integration coverage for point reads, aggregates, mutations, and stream paths
- [x] 9.4 Extend parity checks so HTTP route registration stays in sync with the shared handler surface

## Implementation Details

See TechSpec section "API Endpoints" and ADR-003/ADR-004. HTTP should be a thin registration layer over the shared handlers; this task should not invent HTTP-only semantics.

### Relevant Files
- `internal/api/httpapi/routes.go` — HTTP route registration for `/api/tasks`, `/api/task-runs`, and `/api/observe`
- `internal/api/httpapi/httpapi_integration_test.go` — end-to-end HTTP transport behavior
- `internal/api/httpapi/transport_parity_integration_test.go` — route parity and transport coverage
- `internal/api/httpapi/helpers_integration_test.go` — shared HTTP integration helpers that may need the new task routes

### Dependent Files
- `internal/api/core/tasks.go` — shared handler behavior used by the HTTP routes
- `internal/api/spec/spec.go` — task_07 defines the documented route surface that HTTP must match
- `web/src/systems/tasks/adapters/tasks-api.ts` — task_13 will call these routes through generated clients

### Related ADRs
- [ADR-003: Add Dedicated Task Live Surfaces Instead of Client-Side Stitching](adrs/adr-003.md) — HTTP must expose the task-native live surface directly
- [ADR-004: Use Observer-Backed Read Models for Dashboard, Inbox, and Aggregate Task Views](adrs/adr-004.md) — HTTP must expose observer-backed dashboard and inbox routes

## Deliverables
- Registered HTTP routes for the expanded task surface
- HTTP integration coverage for new task point reads, aggregates, and mutations **(REQUIRED)**
- HTTP parity coverage for the expanded task route family **(REQUIRED)**
- Route registration aligned with the documented OpenAPI surface **(REQUIRED)**
- No HTTP-only task semantics outside the shared handler layer

## Tests
- Unit tests:
  - [x] HTTP route registration includes the new task, task-run, approval, triage, and observe-task endpoints
  - [x] Stream routes are registered with the expected path family, HTTP method, and handler binding
  - [x] Route groups keep actor or workspace middleware aligned with the shared task handlers for point reads and aggregate reads
  - [x] Route registration rejects duplicate or conflicting task path definitions that would cause drift from the documented surface
- Integration tests:
  - [x] HTTP integration tests cover enriched list/detail reads, publish, dashboard, inbox, and run-detail routes against real handler wiring
  - [x] HTTP integration tests cover approval and triage mutations with the expected success and domain-error status codes
  - [x] HTTP integration tests cover task-live routes such as timeline, tree, and stream availability including SSE content type
  - [x] HTTP parity tests fail if the registered route surface drifts from the shared handler contract or documented endpoint family
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80% for modified HTTP transport files
- The daemon exposes the expanded task surface cleanly over HTTP
- Frontend adapter work can target stable, documented HTTP endpoints
