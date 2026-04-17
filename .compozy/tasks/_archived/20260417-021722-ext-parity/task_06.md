---
status: completed
title: "Expose UDS-first resource CRUD APIs"
type: backend
complexity: high
dependencies:
  - task_01
  - task_02
  - task_03
---

# Task 06: Expose UDS-first resource CRUD APIs

## Overview

Add the canonical operator-facing CRUD surface for desired-state resources using the daemon's local control plane first. This task makes resource writes and reads consumable over shared API contracts while explicitly keeping HTTP mutation routes disabled until operator auth middleware exists.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
1. MUST add canonical resource DTOs and shared handler semantics for `GET`, `PUT`, and `DELETE` over `/api/resources` as described in the TechSpec "API Endpoints" section.
2. MUST expose the mutating resource API on UDS first and keep HTTP mutation routes disabled until explicit operator auth middleware is present for those endpoints.
3. MUST map resource validation, scope, conflict, payload, and rate-limit failures to the specified status codes and error payloads consistently across transports.
4. MUST keep operational runtime endpoints family-specific and avoid pushing runs, health, or delivery state into the generic resource CRUD surface.
</requirements>

## Subtasks

- [x] 6.1 Add shared resource DTOs and error mapping in the API contract and core layers
- [x] 6.2 Add UDS route wiring and handler support for list, get, put, and delete resource operations
- [x] 6.3 Gate HTTP mutation routes behind explicit operator-auth availability and keep them disabled otherwise
- [x] 6.4 Add transport coverage for CRUD success paths, conflict paths, and disabled HTTP mutation behavior

## Implementation Details

Follow the TechSpec sections "API Endpoints", "Authority and Validation Rules", and "Integration Points". This task should expose the generic desired-state transport surface only; family-specific runtime actions such as automation runs, hook events, and bridge delivery remain outside this CRUD layer.

### Relevant Files

- `internal/api/contract/` — Add resource CRUD DTOs, filters, and response shapes shared by transports
- `internal/api/core/` — Map resource errors, validation failures, and status codes consistently across transports
- `internal/api/udsapi/routes.go` — Mount the new resource CRUD routes on the local operator control plane
- `internal/api/httpapi/routes.go` — Keep HTTP mutation routes disabled unless operator auth middleware is explicitly available
- `internal/api/spec/spec.go` — Keep shared API documentation aligned with the new resource surface and gating rules

### Dependent Files

- `internal/api/udsapi/handlers_test.go` — UDS handler coverage later depends on the shared resource DTOs and error mapping
- `internal/api/httpapi/handlers_error_test.go` — HTTP behavior needs coverage proving mutation routes remain unavailable without auth middleware
- `internal/daemon/boot.go` — Later daemon composition wiring depends on the canonical API handlers being present

### Related ADRs

- [ADR-005: Make Resource Access Server-Authoritative](adrs/adr-005.md) — Defines the operator-only mutation surface and read restrictions
- [ADR-006: Use a Topology-Aware Reconcile Driver](adrs/adr-006.md) — Post-write success paths must feed the shared reconcile runtime
- [ADR-007: Use Optimistic Concurrency and Serialized Source Snapshots](adrs/adr-007.md) — Drives expected-version and conflict behavior for resource CRUD

## Deliverables

- Shared resource CRUD DTOs and transport-independent error mapping
- UDS routes and handlers for list, get, put, and delete resource operations
- Explicit HTTP route gating that leaves mutation endpoints disabled without operator auth middleware
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration tests for UDS CRUD behavior and HTTP mutation gating **(REQUIRED)**

## Tests

- Unit tests:
  - [x] invalid scope, invalid kind-specific spec, stale `expected_version`, oversized payload, and rate-limit failures map to 422, 409, 413, and 429 consistently
  - [x] handler parsing preserves `expected_version`, scope, and filter semantics for list, get, put, and delete
  - [x] HTTP route registration refuses to expose mutating resource handlers when operator auth middleware is absent
  - [x] operational runtime endpoints remain family-specific and are not routed through generic resource CRUD handlers
- Integration tests:
  - [x] UDS `PUT /api/resources/:kind/:id` creates and updates records with correct 200 or 201 semantics
  - [x] UDS `DELETE /api/resources/:kind/:id` rejects stale versions and succeeds only with the current version
  - [x] HTTP `/api/resources` mutation routes stay unavailable when the server is started without operator auth middleware
- Test coverage target: >=80%
- All tests must pass

## Success Criteria

- All tests passing
- Test coverage >=80%
- Operators can manage desired-state resources through one canonical UDS API surface
- HTTP mutation exposure remains explicitly blocked until operator auth exists instead of silently opening a new write path
