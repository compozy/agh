---
status: completed
title: Expose channel management over shared API contract, HTTP/UDS routes, and OpenAPI
type: backend
complexity: high
dependencies:
  - task_07
---

# Task 08: Expose channel management over shared API contract, HTTP/UDS routes, and OpenAPI

## Overview

Expose the daemon-owned channel subsystem through the shared API contract and both transport servers so operators and future agents can manage channel instances through stable endpoints. This task covers the DTOs, handlers, routes, and OpenAPI output for the channel management surface described in the TechSpec.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
1. MUST add shared API DTOs for channel instance lifecycle, route inspection, and test-delivery operations to `internal/api/contract`.
2. MUST add HTTP and UDS handlers and route registration for the `/api/channels` surface described in the TechSpec, including list, create, get, patch, enable, disable, restart, routes, and test-delivery.
3. MUST update OpenAPI generation so the new channel endpoints and payloads appear in the generated spec consumed by downstream clients.
4. SHOULD map transport DTOs directly to the core channel types introduced in earlier tasks instead of inventing transport-local business models.
</requirements>

## Subtasks
- [x] 8.1 Add shared contract DTOs for channel instances, routes, and test-delivery requests
- [x] 8.2 Add HTTP handlers and route registration for `/api/channels`
- [x] 8.3 Add matching UDS handlers and route registration for `/api/channels`
- [x] 8.4 Update OpenAPI generation and transport tests for the new channel surface

## Implementation Details

Follow the TechSpec sections "HTTP / UDS API", "Impact Analysis", and "Monitoring and Observability". Keep the transport surface aligned with the daemon-owned runtime from task 07 and avoid embedding protocol-specific channel logic in handlers.

### Relevant Files
- `internal/api/contract/contract.go` — Shared request and response DTOs belong here
- `internal/api/contract/responses.go` — Shared transport response helpers may need to expand for channel payloads
- `internal/api/httpapi/routes.go` — HTTP route registration must add the new `/api/channels` group
- `internal/api/udsapi/routes.go` — UDS route registration must mirror the HTTP channel surface
- `internal/api/spec/spec.go` — OpenAPI generation must include the new endpoints and payloads

### Dependent Files
- `internal/cli/client.go` — CLI methods added later depend on the shared transport surface introduced here
- `internal/observe/health.go` — Transport handlers will later expose channel health details based on observability changes
- `internal/api/httpapi/handlers_test.go` — Existing transport coverage patterns should be extended for channel routes and handler behavior

### Related ADRs
- [ADR-005: Hybrid Channel Substrate with Extension-Based Platform Adapters](adrs/adr-005.md) — The daemon-owned substrate must be exposed as a first-class operational API
- [ADR-006: Core-Owned Channel Registry, Scoped Instances, and Policy-Driven Routing](adrs/adr-006.md) — Transport handlers operate on registry-owned instances and routes

## Deliverables
- Shared transport DTOs for channel lifecycle, route inspection, and test delivery
- HTTP and UDS `/api/channels` routes and handlers
- OpenAPI spec updates for the new channel surface
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration tests for HTTP and UDS channel flows **(REQUIRED)**

## Tests
- Unit tests:
  - [x] Channel request DTO validation rejects malformed scope, workspace, and routing-policy payloads
  - [x] Channel route response payloads serialize peer, thread, and group fields consistently
  - [x] Test-delivery request mapping preserves the typed `DeliveryTarget` shape rather than flattening it into platform strings
- Integration tests:
  - [x] `POST /api/channels` creates a channel instance and returns the persisted instance payload
  - [x] `GET /api/channels/:id/routes` returns only the route set owned by the requested channel instance
  - [x] `POST /api/channels/:id/test-delivery` exercises outbound target resolution without requiring a live platform adapter
  - [x] The UDS transport mirrors the same create, get, and route-inspection behavior as the HTTP surface
  - [x] OpenAPI generation includes the new `/api/channels` endpoints and schemas
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- Operators and future agents can manage channels through stable HTTP and UDS endpoints
- The generated OpenAPI spec includes the channel management surface
