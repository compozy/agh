---
status: completed
title: Shared task handlers in api/core
type: backend
complexity: high
dependencies:
  - task_07
---

# Task 08: Shared task handlers in api/core

## Overview

Implement the shared handler layer for the expanded tasks surface so HTTP and UDS can expose the same semantics without drifting. This task translates the new task-manager, task-live, and observer-backed task reads into one consistent `api/core` surface.

<critical>
- ALWAYS READ `_techspec.md`, ADRs, and `task_07.md` before adding handler behavior
- REFERENCE TECHSPEC sections "System Architecture", "API Endpoints", and "Response and status conventions"
- FOCUS ON "WHAT" — add shared handler behavior, parsers, and conversions, not transport-specific route tables
- MINIMIZE CODE — keep all parsing, error mapping, and response shaping in `api/core` instead of duplicating logic per transport
- TESTS REQUIRED — query parsing, SSE behavior, conversions, and status mapping all need coverage
- GREENFIELD: HTTP e UDS precisam compartilhar o mesmo comportamento de tasks; nao duplique parser/conversion logic em dois transportes
</critical>

<requirements>
- MUST add shared handlers for publish, run detail, timeline, stream, tree, dashboard, inbox, approval, and triage actions
- MUST extend `api/core` interfaces so handlers can depend on the task manager, task-live service, and observer-backed task reads without transport leaks
- MUST parse the new task query parameters and map them to the shared contract types consistently
- MUST use the existing SSE helper surface for task-native streaming instead of hand-rolling stream responses per transport
- MUST preserve existing error/status conventions for validation, not found, conflict, and service-unavailable cases
- SHOULD keep conversions between domain views and public payloads isolated and reusable
</requirements>

## Subtasks
- [x] 8.1 Extend `api/core` interfaces for task-live and aggregate task reads
- [x] 8.2 Add shared handlers for the new task point reads, aggregate reads, and mutations
- [x] 8.3 Extend parsers and conversions for the new task queries and payloads
- [x] 8.4 Wire task-native streaming through the shared SSE helper path
- [x] 8.5 Add handler, parser, and conversion coverage for the expanded surface

## Implementation Details

See TechSpec sections "System Architecture", "API Endpoints", and ADR-003/ADR-004. `api/core` is the one place where task-manager, task-live, and observer-backed surfaces should be normalized for transport exposure.

### Relevant Files
- `internal/api/core/tasks.go` — existing shared task handlers that must grow with the new task surface
- `internal/api/core/interfaces.go` — shared service interfaces for handlers
- `internal/api/core/parsers.go` — query decoding and validation for transport inputs
- `internal/api/core/conversions.go` — shared response shaping for public payloads
- `internal/api/core/sse.go` — helper surface for task-native streaming responses
- `internal/api/core/tasks_test.go` — core handler behavior coverage
- `internal/api/core/tasks_integration_test.go` — integration-level task handler coverage
- `internal/api/testutil/apitest.go` — stubs and helpers used by task handler tests

### Dependent Files
- `internal/api/httpapi/routes.go` — task_09 will register the shared handlers
- `internal/api/udsapi/routes.go` — task_10 will register the shared handlers
- `internal/api/httpapi/transport_parity_integration_test.go` — should verify transport parity on the expanded task surface
- `internal/api/udsapi/transport_parity_integration_test.go` — should verify transport parity on the expanded task surface

### Related ADRs
- [ADR-003: Add Dedicated Task Live Surfaces Instead of Client-Side Stitching](adrs/adr-003.md) — Shared handlers must expose task-native live APIs cleanly
- [ADR-004: Use Observer-Backed Read Models for Dashboard, Inbox, and Aggregate Task Views](adrs/adr-004.md) — Shared handlers must normalize observer-backed aggregate reads

## Deliverables
- Shared `api/core` handlers, parsers, and conversions for the expanded task surface
- Task-native SSE handler behavior built on the existing shared stream helpers
- Unit tests with >=80% coverage for handler parsing and conversion behavior **(REQUIRED)**
- Integration tests proving the shared task handlers against stubbed services **(REQUIRED)**
- One canonical task API behavior surface reused by both transports

## Tests
- Unit tests:
  - [x] New task query parameters parse into the expected contract and domain-query structures for list, timeline, tree, dashboard, and inbox reads
  - [x] Handler conversions return the expected enriched task, inbox, dashboard, timeline, tree, and run-detail payloads with optional fields preserved correctly
  - [x] Publish, approval, rejection, and triage actions map domain errors to the expected HTTP or UDS status semantics
  - [x] Actor and workspace context extraction is forwarded consistently to task, observer, and live-read services
  - [x] Task-native stream handlers emit the expected SSE framing, headers, and validation behavior for missing or invalid identifiers
- Integration tests:
  - [x] Shared handler tests cover list, detail, run-detail, dashboard, inbox, and live reads against stubbed services
  - [x] Mutation handlers cover publish, approve, reject, archive, dismiss, and mark-read flows against stubbed services
  - [x] Stream handlers integrate with the shared SSE helper path without transport-specific divergence or missing flush behavior
  - [x] Shared handler contract tests fail if service interfaces or response shapes drift from the transport-facing task surface
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80% for modified `api/core` files
- HTTP and UDS can expose the expanded task surface through one shared handler layer
- Transport wiring tasks can register routes without needing to re-implement task behavior
