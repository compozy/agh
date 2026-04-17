---
status: pending
title: UDS task transport and parity coverage
type: backend
complexity: medium
dependencies:
  - task_08
---

# Task 10: UDS task transport and parity coverage

## Overview

Mirror the expanded task surface over the daemon’s UDS transport and keep it behaviorally aligned with HTTP. This task ensures CLI and local tool consumers receive the same task capabilities, not a smaller or drifting subset.

<critical>
- ALWAYS READ `_techspec.md`, `task_07.md`, `task_08.md`, and `task_09.md` before wiring UDS routes
- REFERENCE TECHSPEC sections "API Endpoints", "Development Sequencing", and "Response and status conventions"
- FOCUS ON "WHAT" — register and verify UDS parity for the shared task surface, not new handler behavior
- MINIMIZE CODE — mirror the shared task surface through route registration and parity tests only
- TESTS REQUIRED — UDS integration and parity coverage must prove there is no transport drift
- GREENFIELD: UDS nao pode ficar atras do HTTP nessa feature; CLI e automacoes locais precisam ver a mesma superficie de tasks
</critical>

<requirements>
- MUST register the expanded task point-read, task-live, and task-observe routes in the UDS transport
- MUST keep UDS route shapes aligned with the shared contracts and HTTP route family
- MUST extend UDS integration tests for the new task endpoints and mutations
- MUST add or update parity checks so route drift between UDS and HTTP is caught automatically
- SHOULD preserve the same path taxonomy under `/api/tasks`, `/api/task-runs`, and `/api/observe/tasks`
</requirements>

## Subtasks
- [ ] 10.1 Register the new UDS task, task-run, and observe-task routes
- [ ] 10.2 Extend UDS integration tests for point reads, aggregates, and mutations
- [ ] 10.3 Extend transport parity checks so UDS and HTTP stay synchronized

## Implementation Details

See TechSpec section "API Endpoints" and ADR-003/ADR-004. UDS exists so local CLI and automation consumers can use the same task surface as the web app; route drift here is a real product bug.

### Relevant Files
- `internal/api/udsapi/routes.go` — UDS route registration for the task surface
- `internal/api/udsapi/udsapi_integration_test.go` — broad UDS integration coverage
- `internal/api/udsapi/transport_parity_integration_test.go` — transport parity verification between UDS and HTTP
- `internal/api/udsapi/handlers_test.go` — route/handler registration expectations

### Dependent Files
- `internal/api/core/tasks.go` — shared handler behavior consumed by UDS
- `internal/api/httpapi/routes.go` — parity target that UDS must stay aligned with
- `internal/cli/client_test.go` — downstream CLI expectations may need updates once the route surface expands

### Related ADRs
- [ADR-003: Add Dedicated Task Live Surfaces Instead of Client-Side Stitching](adrs/adr-003.md) — UDS must expose the task-native live surface, not only the older point reads
- [ADR-004: Use Observer-Backed Read Models for Dashboard, Inbox, and Aggregate Task Views](adrs/adr-004.md) — UDS must expose observer-backed dashboard and inbox routes as well

## Deliverables
- Registered UDS routes for the expanded task surface
- UDS integration coverage for the new task routes and mutations **(REQUIRED)**
- Updated transport parity coverage that includes the new task endpoints **(REQUIRED)**
- Consistent task behavior across HTTP and UDS **(REQUIRED)**

## Tests
- Unit tests:
  - [ ] UDS route registration includes the new task, task-run, approval, triage, and observe-task endpoints
  - [ ] Handler registration expectations stay aligned with the shared task surface for reads, mutations, and aggregate views
  - [ ] UDS route metadata keeps method or operation naming stable enough for parity assertions against HTTP
- Integration tests:
  - [ ] UDS integration tests cover enriched list/detail reads, publish, dashboard, inbox, and run-detail routes against real handler wiring
  - [ ] UDS integration tests cover approval and triage mutations with the same payload and status semantics expected by HTTP
  - [ ] UDS integration tests cover task-live routes such as timeline, tree, and stream-path availability for the expanded task surface
  - [ ] Transport parity tests confirm UDS and HTTP expose the same expanded task route family without missing or extra endpoints
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80% for modified UDS transport files
- UDS exposes the same expanded task surface as HTTP
- CLI and local agent tooling can rely on transport parity for the full tasks feature
