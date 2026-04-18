---
status: completed
title: UDS settings transport and parity coverage
type: backend
complexity: high
dependencies:
  - task_05
---

# Task 07: UDS settings transport and parity coverage

## Overview

Mirror the settings API surface on UDS so the CLI and trusted local automation have full privileged access to the same settings contract. This task keeps UDS aligned with HTTP for route shape and payload semantics while preserving UDS as the authoritative local privileged transport.

<critical>
- ALWAYS READ `_techspec.md` and ADRs before starting (`_prd.md` is absent; requirements come from the TechSpec)
- REFERENCE TECHSPEC sections "API Endpoints", "Transport and security policy", and "Testing Approach"
- FOCUS ON "WHAT" — keep UDS semantically aligned with HTTP without copying transport-specific policy
- MINIMIZE CODE — reuse shared core handlers and existing UDS route organization
- TESTS REQUIRED — parity, route inventory, and settings action coverage must be explicit
- GREENFIELD: não aceitar deriva entre HTTP e UDS; o contrato de settings precisa ser único
</critical>

<requirements>
- MUST register the full `/api/settings/*` namespace on UDS with the same payloads as HTTP
- MUST expose restart actions and status polling over UDS
- MUST preserve UDS as a privileged local transport without applying the HTTP loopback restriction to it
- MUST keep UDS extension routes aligned with the HTTP-visible settings dependency surface
- MUST add transport-parity coverage so route or payload drift is caught automatically
- SHOULD follow existing UDS route grouping and handler wrapper patterns
</requirements>

## Design References

This task is foundational — the UDS transport mirrors the HTTP settings contract for CLI parity and underpins all 10 settings screens. See `_techspec.md` → *Design References* for the full 10-artboard table and the task-to-screen mapping.

## Subtasks

- [x] 7.1 Register all settings routes on UDS, including restart and log-tail endpoints
- [x] 7.2 Align UDS extension and settings route coverage with the HTTP-visible contract
- [x] 7.3 Reuse shared `api/core` handlers without transport-specific payload forks
- [x] 7.4 Extend UDS route inventory and transport parity tests for settings
- [x] 7.5 Add UDS handler tests for restart and collection mutation flows

## Implementation Details

See TechSpec sections "API Endpoints", "Transport and security policy", and ADR-001/ADR-004. This task should not re-implement business logic already present in `api/core`; it should mirror the route and payload surface over UDS and extend parity protections.

### Relevant Files

- `internal/api/udsapi/routes.go` — UDS route registration point for the new settings namespace
- `internal/api/udsapi/server.go` — UDS transport wiring and setup
- `internal/api/udsapi/handlers.go` — transport wrappers around shared core handlers
- `internal/api/udsapi/transport_parity_integration_test.go` — existing parity coverage that should include settings
- `internal/api/udsapi/server_test.go` — likely place for UDS route and handler integration checks

### Dependent Files

- `internal/api/udsapi/handlers_test.go` — should add settings happy-path and error-path tests
- `internal/api/httpapi/transport_parity_integration_test.go` — should stay aligned with UDS after this task
- `internal/cli/` — future CLI or local automation can rely on the new privileged settings UDS surface
- `web/` — does not use UDS directly but benefits from parity guarantees

### Related ADRs

- [ADR-001: Use a consolidated settings namespace with a dedicated settings shell](adrs/adr-001.md) — Requires one coherent settings surface across transports
- [ADR-004: Restrict HTTP settings mutations to loopback-bound servers in v1](adrs/adr-004.md) — Keeps UDS as the authoritative privileged local transport

## Deliverables

- Full UDS settings route registration aligned with the shared contract
- Restart and collection mutation support over UDS
- Transport-parity and UDS handler coverage for settings **(REQUIRED)**
- Unit tests with >=80% coverage for modified `internal/api/udsapi` surface **(REQUIRED)**
- Integration tests that verify UDS behavior remains aligned with HTTP semantics **(REQUIRED)**

## Tests

- Unit tests:
  - [x] All required UDS settings routes are registered with the expected verbs and wrappers
  - [x] UDS restart actions and status polling return the same payload shapes as HTTP
  - [x] UDS collection mutation handlers surface the same validation and conflict behavior as HTTP
  - [x] UDS route plumbing preserves `scope`, `workspace_id`, and `target` query semantics for MCP server operations
  - [x] UDS settings routes reuse shared `api/core` handlers instead of introducing transport-specific DTO forks
- Integration tests:
  - [x] Transport parity tests verify the full settings route inventory across HTTP and UDS
  - [x] UDS settings mutations execute without the HTTP loopback restriction
  - [x] UDS extension route coverage stays aligned with the settings surface requirements
  - [x] Workspace-scoped `mcp-servers` reads and writes succeed over UDS with the same response shapes exposed on HTTP
- Test coverage target: >=80%
- All tests must pass

## Success Criteria

- All tests passing
- Test coverage >=80% for modified `internal/api/udsapi`
- UDS exposes the full settings contract for local privileged tooling
- Transport parity tests prevent HTTP and UDS settings drift
