---
status: pending
title: HTTP settings transport and loopback mutation policy
type: backend
complexity: high
dependencies:
  - task_05
---

# Task 06: HTTP settings transport and loopback mutation policy

## Overview

Expose the settings contract over the HTTP transport while enforcing the v1 security posture for mutating routes. This task registers `/api/settings/*`, ensures the required extension HTTP parity for the Hooks & Extensions screen, and returns `403` for HTTP mutations when the daemon is not loopback-bound.

<critical>
- ALWAYS READ `_techspec.md` and ADRs before starting (`_prd.md` is absent; requirements come from the TechSpec)
- REFERENCE TECHSPEC sections "API Endpoints", "Transport and security policy", and "Testing Approach"
- FOCUS ON "WHAT" — register the HTTP surface and apply transport policy consistently
- MINIMIZE CODE — reuse shared `api/core` handlers and existing HTTP middleware patterns
- TESTS REQUIRED — route inventory, loopback gating, and HTTP extension parity all need coverage
- GREENFIELD: não abrir superfície de admin remoto por acidente; bind não-loopback deve bloquear mutações
</critical>

<requirements>
- MUST register all read and mutation routes under `/api/settings/*` on HTTP
- MUST expose required `/api/extensions` HTTP routes for the Hooks & Extensions settings screen
- MUST enforce loopback-only access for HTTP settings mutations, restart actions, and HTTP extension mutations
- MUST return `403` for blocked mutation attempts on non-loopback HTTP binds
- MUST keep response payloads and route shapes aligned with the shared contract and UDS behavior
- SHOULD keep route registration and policy enforcement explicit so route inventory tests can catch drift
</requirements>

## Design References

This task is foundational — the HTTP transport carries every settings read/write used by the web UI, so all 10 settings screens depend on it. See `_techspec.md` → *Design References* for the full 10-artboard table and the task-to-screen mapping.

## Subtasks

- [ ] 6.1 Register the full HTTP settings route namespace and restart/status endpoints
- [ ] 6.2 Register HTTP-visible extension routes required by the settings surface
- [ ] 6.3 Enforce loopback-only gating for mutating settings and extension routes
- [ ] 6.4 Add HTTP route inventory coverage for settings and extension parity
- [ ] 6.5 Add handler and integration tests for `403` policy behavior on non-loopback binds

## Implementation Details

See TechSpec sections "API Endpoints", "Transport and security policy", and ADR-004. This task should reuse `api/core` handlers from task_05 and keep security policy at the HTTP transport boundary rather than inside the settings service.

### Relevant Files

- `internal/api/httpapi/routes.go` — HTTP route registration point for the new settings namespace
- `internal/api/httpapi/handlers.go` — transport wrapper around shared core handlers
- `internal/api/httpapi/middleware.go` — natural place for loopback-bound mutation enforcement if not done in route wrappers
- `internal/api/httpapi/server.go` — source of bind-host context used by the policy
- `internal/api/httpapi/transport_parity_integration_test.go` — existing parity-style coverage to extend

### Dependent Files

- `internal/api/httpapi/handlers_test.go` — should add settings route and forbidden-path coverage
- `internal/api/httpapi/server_test.go` — should verify loopback and non-loopback bind behavior
- `web/src/systems/settings/adapters/settings-api.ts` — will consume these HTTP routes in task_09
- `web/src/routes/_app/settings/*.tsx` — will rely on the registered HTTP settings namespace

### Related ADRs

- [ADR-001: Use a consolidated settings namespace with a dedicated settings shell](adrs/adr-001.md) — Defines the HTTP namespace the web app consumes
- [ADR-004: Restrict HTTP settings mutations to loopback-bound servers in v1](adrs/adr-004.md) — Defines the mutation security policy

## Deliverables

- HTTP route registration for `/api/settings/*` plus required `/api/extensions` parity
- Loopback-only gating for HTTP settings mutations, restart actions, and extension mutations
- Route inventory and integration coverage for HTTP settings behavior **(REQUIRED)**
- Unit tests with >=80% coverage for modified HTTP routing and policy code **(REQUIRED)**
- Integration tests that exercise non-loopback `403` behavior and loopback success paths **(REQUIRED)**

## Tests

- Unit tests:
  - [ ] All required HTTP settings routes are registered with the expected verbs
  - [ ] Required `/api/extensions` routes are available on HTTP for the settings screen
  - [ ] Non-loopback HTTP binds return `403` for settings mutations and restart actions
  - [ ] Read-only settings routes remain available on HTTP regardless of bind host
- Integration tests:
  - [ ] Loopback-bound HTTP server allows settings mutation requests to reach shared handlers
  - [ ] Non-loopback HTTP server blocks settings and extension mutations while preserving read routes
  - [ ] HTTP transport parity tests catch route drift relative to UDS
- Test coverage target: >=80%
- All tests must pass

## Success Criteria

- All tests passing
- Test coverage >=80% for modified `internal/api/httpapi`
- The web app can consume the full settings read surface over HTTP
- HTTP no longer exposes privileged settings or extension mutations when bound beyond loopback
