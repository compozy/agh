---
status: pending
title: Task API contracts and OpenAPI codegen
type: backend
complexity: critical
dependencies:
  - task_03
  - task_04
  - task_05
  - task_06
---

# Task 07: Task API contracts and OpenAPI codegen

## Overview

Expose the expanded task read/write surface through authoritative shared contracts and regenerated OpenAPI types. This task is the contract boundary between backend work and the web tasks system: every new task, observe, and stream payload must be documented here before frontend integration begins.

<critical>
- ALWAYS READ `_techspec.md`, ADRs, and `task_03.md` through `task_06.md` before changing public contracts
- REFERENCE TECHSPEC sections "API Endpoints", "Response and status conventions", and "Development Sequencing"
- FOCUS ON "WHAT" — define the contract surface and generated type authority, not handler implementation
- MINIMIZE CODE — keep payloads explicit and shared; do not fork separate HTTP and frontend-only DTOs
- TESTS REQUIRED — spec generation, payload shape, and stream documentation need coverage
- GREENFIELD: o OpenAPI precisa ser a fonte autoritativa; nao deixe endpoints novos existirem so no router ou em tipos manuais do frontend
</critical>

<requirements>
- MUST add shared contracts for enriched task list/detail reads, draft publication, timeline, stream, tree, run detail, dashboard, inbox, approval, and triage mutations
- MUST document all new task and observe endpoints in `internal/api/spec/spec.go`, including `text/event-stream` for task-native streaming
- MUST regenerate the frontend OpenAPI types so `web/src/generated/agh-openapi.d.ts` reflects the new contracts
- MUST keep HTTP and UDS surfaces aligned through one shared contract vocabulary
- MUST avoid relying on undocumented session-stream fallbacks for task-live behavior in generated frontend types
- SHOULD keep contract naming aligned with existing `contract.Task*` payload conventions
</requirements>

## Subtasks
- [ ] 7.1 Add the new shared task and observe payload/request types
- [ ] 7.2 Extend the OpenAPI spec with the new endpoints, payloads, and stream documentation
- [ ] 7.3 Regenerate the frontend OpenAPI types and verify codegen consistency
- [ ] 7.4 Add spec and contract tests for the new task surface

## Implementation Details

See TechSpec sections "API Endpoints", "Response and status conventions", and ADR-003/ADR-004. This task is complete only when the generated frontend types and the documented spec agree with the backend intent; route registration without contract coverage is not enough.

### Relevant Files
- `internal/api/contract/tasks.go` — shared task payloads, requests, and response envelopes for the expanded surface
- `internal/api/contract/contract.go` — shared aggregate/observe payload definitions that may need task dashboard/inbox additions
- `internal/api/spec/spec.go` — OpenAPI endpoint and schema definitions for the new task surface
- `internal/api/spec/spec_test.go` — spec-level regression coverage
- `web/src/generated/agh-openapi.d.ts` — generated frontend types that must be refreshed by codegen
- `Makefile` — exposes `make codegen` and `make codegen-check`, which should remain authoritative for regeneration

### Dependent Files
- `internal/api/core/tasks.go` — task_08 will implement handlers against these contracts
- `internal/api/httpapi/routes.go` — task_09 will register the documented endpoints
- `internal/api/udsapi/routes.go` — task_10 will mirror the documented endpoints
- `internal/extension/host_api_tasks.go` — task_11 may reuse the new payload vocabulary
- `web/src/lib/api-contract.ts` — frontend contract consumers rely on the regenerated type surface

### Related ADRs
- [ADR-002: Expand the Task Domain for Paper-Parity Semantics](adrs/adr-002.md) — Public contracts must expose the first-class task semantics added to the domain
- [ADR-003: Add Dedicated Task Live Surfaces Instead of Client-Side Stitching](adrs/adr-003.md) — Requires contract support for task timeline, stream, tree, and run detail
- [ADR-004: Use Observer-Backed Read Models for Dashboard, Inbox, and Aggregate Task Views](adrs/adr-004.md) — Requires explicit dashboard and inbox contract surfaces

## Deliverables
- Shared contract payloads and request types for the expanded task surface
- OpenAPI coverage for all new task and observe endpoints, including task-native SSE **(REQUIRED)**
- Regenerated `web/src/generated/agh-openapi.d.ts` aligned with the backend spec **(REQUIRED)**
- Contract and spec tests with >=80% coverage for modified files **(REQUIRED)**
- No undocumented task endpoints left outside the generated frontend type surface

## Tests
- Unit tests:
  - [ ] Contract payloads serialize the expected enriched task list, task detail, dashboard, inbox, timeline, tree, and run-detail fields including optional references
  - [ ] Publish, approval, and triage request or response contracts declare the expected required and optional fields without metadata fallbacks
  - [ ] Stream endpoint documentation declares the expected `text/event-stream` media type, path params, and event payload shape
  - [ ] Error contracts and status-code documentation remain consistent for `404`, `409`, `422`, and transport-safe failure cases across the new routes
  - [ ] Generated type references and operation identifiers remain stable for the new task operations
- Integration tests:
  - [ ] `make codegen` or `make codegen-check` succeeds with the updated spec and produces no unexpected diff outside generated artifacts
  - [ ] Spec tests confirm the new task, task-run, and observe-task endpoints are present, tagged correctly, and well-formed
  - [ ] Generated client or type artifacts compile against the updated contracts without manual patching or type escapes
  - [ ] Operation IDs, route paths, and method combinations remain unique so codegen consumers do not drift
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80% for modified contract/spec files
- The expanded task surface is fully represented in shared contracts and generated frontend types
- Frontend implementation can begin against authoritative OpenAPI-backed types instead of handwritten DTO assumptions
