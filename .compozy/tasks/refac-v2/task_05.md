---
status: pending
title: Re-root HTTP and UDS transports plus shared API test utilities
type: refactor
complexity: critical
dependencies:
  - task_03
  - task_04
---

# Task 05: Re-root HTTP and UDS transports plus shared API test utilities

## Overview
This task completes the API subtree migration by moving HTTP, UDS, and shared API testing support under `internal/api/*`. It is the highest-risk API task because it touches runtime transports, route registration, and the integration suites that prove the public daemon API remains stable.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- `internal/httpapi`, `internal/udsapi`, and `internal/apitest` MUST be re-rooted under the `internal/api/` subtree.
- Existing route paths, request semantics, status codes, SSE behavior, and static HTTP behavior MUST remain unchanged.
- Shared API test harnesses MUST move into `internal/api/testutil` so transport and CLI tests can reuse them from the new subtree.
- This task MUST run both `make verify` and `make test-integration` because it changes runtime transport surfaces.
</requirements>

## Subtasks
- [ ] 5.1 Re-root HTTP transport code into `internal/api/httpapi` while preserving server lifecycle and static asset behavior.
- [ ] 5.2 Re-root UDS transport code into `internal/api/udsapi` while preserving route registration and socket lifecycle behavior.
- [ ] 5.3 Re-root `internal/apitest` into `internal/api/testutil` and update transport and CLI test imports.
- [ ] 5.4 Remove temporary package bridges introduced for the move and keep the runtime API surface stable.

## Implementation Details
Use the TechSpec `Component Overview`, `API Endpoints`, and `Testing Approach` sections. Preserve the external route surface exactly. Keep transport-only behavior local to the transport packages even after the move.

### Relevant Files
- `internal/httpapi/server.go` — Owns HTTP server lifecycle, route setup, and static asset behavior.
- `internal/httpapi/prompt.go` — Contains HTTP-only prompt streaming behavior that must remain transport-local.
- `internal/udsapi/server.go` — Owns UDS server lifecycle.
- `internal/udsapi/routes.go` — Owns the shared route registrations that must preserve path compatibility.
- `internal/apitest/apitest.go` — Current shared API test harness to re-root into `internal/api/testutil`.

### Dependent Files
- `internal/httpapi/httpapi_integration_test.go` — Must prove HTTP runtime behavior remains stable after re-rooting.
- `internal/httpapi/handlers_test.go` — Must continue validating the registered HTTP route surface.
- `internal/udsapi/udsapi_integration_test.go` — Must prove UDS runtime behavior remains stable after re-rooting.
- `internal/udsapi/handlers_test.go` — Must continue validating the registered UDS route surface.
- `internal/cli/cli_integration_test.go` — Consumes shared API test infrastructure and must still pass after the move.

### Related ADRs
- [ADR-001: Adopt a Broad Package-Graph Reorganization for Refac V2](../adrs/adr-001.md) — Requires the explicit `internal/api/*` subtree.
- [ADR-004: Use Phased Cutovers with Same-Phase Bridge Removal and Layered Verification](../adrs/adr-004.md) — Requires same-phase bridge removal plus verify and integration gates.

## Deliverables
- HTTP and UDS transport packages re-rooted into `internal/api/httpapi` and `internal/api/udsapi`.
- Shared API test support re-rooted into `internal/api/testutil`.
- Updated imports across transport tests and CLI integration tests.
- `make verify` and `make test-integration` passing for the moved API runtime surface.

## Tests
- Unit tests:
  - [ ] HTTP route registration still exposes the same session, workspace, agent, observe, memory, and daemon endpoints.
  - [ ] UDS route registration still exposes the same endpoint set and handler bindings.
  - [ ] Shared API test helpers continue building valid request harnesses and SSE assertions after re-rooting.
- Integration tests:
  - [ ] HTTP integration suite still passes for session lifecycle, memory endpoints, static assets, and streaming behavior.
  - [ ] UDS integration suite still passes for session lifecycle, memory endpoints, and streaming behavior.
  - [ ] CLI integration suite still passes when using the re-rooted shared API test utilities.
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- The API subtree lives under `internal/api/*` without route or transport regressions
- No temporary re-export bridges remain from the re-root phase
