---
status: pending
title: Shared settings handlers in api/core
type: backend
complexity: high
dependencies:
  - task_02
  - task_03
  - task_04
---

# Task 05: Shared settings handlers in api/core

## Overview

Wire the settings service into the transport-neutral API core so HTTP and UDS can share one implementation path for reads, mutations, restart actions, and polling. This task centralizes request parsing, service invocation, and status mapping without leaking transport-specific policy into the service layer.

<critical>
- ALWAYS READ `_techspec.md` and ADRs before starting (`_prd.md` is absent; requirements come from the TechSpec)
- REFERENCE TECHSPEC sections "Core Interfaces", "API Endpoints", and "Error handling conventions"
- FOCUS ON "WHAT" — centralize settings request handling in `api/core`, not transport policy
- MINIMIZE CODE — reuse existing core patterns for parsing, conversions, and error mapping
- TESTS REQUIRED — invalid payloads, unsupported scope, restart action, and status polling need coverage
- GREENFIELD: prefer explicit error mapping and typed payloads over generic handler reuse that hides behavior
</critical>

<requirements>
- MUST add a settings service dependency to `internal/api/core`
- MUST implement transport-neutral handlers for section reads, section updates, collection list/get/put/delete, restart action trigger, restart status polling, and log-tail metadata or stream plumbing expected by transports
- MUST map validation, not-found, conflict, forbidden, and internal errors to the TechSpec status conventions
- MUST keep transport-specific loopback policy out of `api/core`; that belongs to HTTP routing and middleware
- MUST support the restart action as an asynchronous `202 Accepted` flow with status URLs and persisted operation ids
- SHOULD use existing parser and payload helper patterns so HTTP and UDS stay behaviorally identical
</requirements>

## Design References

This task is foundational — the shared `api/core` handlers fan out to every settings page and collection page. See `_techspec.md` → *Design References* for the full 10-artboard table and the task-to-screen mapping.

## Subtasks

- [ ] 5.1 Add settings service interfaces and dependencies to `internal/api/core`
- [ ] 5.2 Implement section and collection handlers using shared contract DTOs
- [ ] 5.3 Implement restart trigger and restart-status polling handlers
- [ ] 5.4 Add settings-specific error mapping and payload validation helpers
- [ ] 5.5 Cover happy-path and error-path behavior with core-level tests

## Implementation Details

See TechSpec sections "Core Interfaces", "API Endpoints", "Response behavior", and "Testing Approach". This task should stop at shared core behavior; route registration and HTTP loopback enforcement are separate transport tasks.

### Relevant Files

- `internal/api/core/interfaces.go` — shared service interface definitions and injected dependencies
- `internal/api/core/handlers.go` — central handler entry points that transports reuse
- `internal/api/core/errors.go` — status-code mapping and structured error translation
- `internal/api/core/payloads.go` — request payload parsing and validation helpers
- `internal/api/core/conversions.go` — shared response shaping utilities

### Dependent Files

- `internal/api/httpapi/handlers.go` — will expose these handlers over HTTP in task_06
- `internal/api/udsapi/handlers.go` — will expose these handlers over UDS in task_07
- `internal/api/core/*_test.go` — should add settings-specific coverage here instead of only at transport level
- `internal/daemon/daemon.go` — must provide the settings service dependency during wiring

### Related ADRs

- [ADR-001: Use a consolidated settings namespace with a dedicated settings shell](adrs/adr-001.md) — Requires a single shared settings API surface
- [ADR-003: Keep settings mutations restart-aware and separate from operational workflows](adrs/adr-003.md) — Defines restart action and mutation result behavior
- [ADR-004: Restrict HTTP settings mutations to loopback-bound servers in v1](adrs/adr-004.md) — Informs forbidden-status mapping even though policy enforcement stays in HTTP

## Deliverables

- Shared `api/core` settings handlers and injected service interfaces
- Error mapping for validation, forbidden, not-found, conflict, and internal settings failures
- Restart action and polling handlers that return the contract defined in task_04 **(REQUIRED)**
- Unit tests with >=80% coverage for new `api/core` settings behavior **(REQUIRED)**
- Integration-style handler tests that verify status mapping and payload shaping **(REQUIRED)**

## Tests

- Unit tests:
  - [ ] Invalid section payloads return `400` with section-specific context
  - [ ] Missing resources return `404` and conflicting names or targets return `409`
  - [ ] Restart action handler returns `202` with operation id, status URL, and active session count
  - [ ] Restart status polling returns the persisted operation record shape from the contract
  - [ ] `MutationResult` responses preserve semantic `write_target` and restart metadata
  - [ ] Collection PUT and DELETE handlers preserve `scope`, `workspace_id`, and `target` semantics when delegating into the settings service
  - [ ] Action-trigger sections map to the correct downstream action handlers instead of being treated as restart-required config saves
- Integration tests:
  - [ ] Shared handlers behave identically across HTTP/UDS shims in core-level tests
  - [ ] Log-tail or observability action plumbing exposes the expected response contract to transports
  - [ ] Restart-trigger and restart-status handlers round-trip the same operation identifiers and polling payloads through transport-facing adapters
- Test coverage target: >=80%
- All tests must pass

## Success Criteria

- All tests passing
- Test coverage >=80% for modified `internal/api/core`
- HTTP and UDS can reuse one settings handler layer without divergent behavior
- Error codes and payload shapes match the TechSpec before transport wiring begins
