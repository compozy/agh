---
status: pending
title: Settings API contract and OpenAPI surface
type: backend
complexity: high
dependencies:
  - task_02
  - task_03
---

# Task 04: Settings API contract and OpenAPI surface

## Overview

Translate the settings service and restart operation model into stable API DTOs and OpenAPI definitions that both transports and the web client can consume. This task closes the contract gap for `/api/settings/*`, restart polling, log tail metadata, and the HTTP-visible extension surface needed by the settings UI.

<critical>
- ALWAYS READ `_techspec.md` and ADRs before starting (`_prd.md` is absent; requirements come from the TechSpec)
- REFERENCE TECHSPEC sections "Core Interfaces", "API Endpoints", and "Impact Analysis"
- FOCUS ON "WHAT" — define stable payloads and route contracts, not handler behavior
- MINIMIZE CODE — keep contract and spec aligned; avoid transport-specific DTO forks
- TESTS REQUIRED — OpenAPI and generated web types must remain synchronized
- GREENFIELD: prefer explicit, typed payloads over loose maps or payload reuse that hides settings semantics
</critical>

<requirements>
- MUST add DTOs for settings sections, collection resources, mutation results, restart actions, and restart status polling
- MUST define the `/api/settings/*` surface in `internal/api/spec` with the same contract expected by both HTTP and UDS
- MUST expose semantic `write_target`, source-precedence metadata, and restart status payloads in the contract
- MUST include HTTP-visible extension routes required by the Hooks & Extensions screen in the OpenAPI surface
- MUST regenerate any checked-in API artifacts or generated web types that depend on the OpenAPI schema
- SHOULD keep route and payload naming aligned with the TechSpec to avoid drift between daemon and web
</requirements>

## Subtasks

- [ ] 4.1 Add settings DTOs and restart payloads in `internal/api/contract`
- [ ] 4.2 Add `/api/settings/*`, restart, and log-tail operations to `internal/api/spec`
- [ ] 4.3 Include HTTP extension surface required by the settings UI in the contract/spec
- [ ] 4.4 Regenerate checked-in OpenAPI artifacts and generated web types
- [ ] 4.5 Add or update spec tests that verify route and schema coverage

## Implementation Details

See TechSpec sections "Core Interfaces", "API Endpoints", "Response behavior", and ADR-001/ADR-004. This task should not implement handlers yet; it should only define shared contract shapes and authoritative API documentation consumed by later backend and frontend tasks.

### Relevant Files

- `internal/api/contract/contract.go` — shared API payload definitions and the natural place for new settings DTOs
- `internal/api/contract/responses.go` — existing response helpers and envelope patterns
- `internal/api/spec/spec.go` — authoritative OpenAPI registration and schema surface
- `internal/api/spec/spec_test.go` — existing spec verification coverage
- `web/src/generated/agh-openapi.d.ts` — generated web types that must remain aligned

### Dependent Files

- `internal/api/core/handlers.go` — will consume these DTOs in task_05
- `internal/api/httpapi/routes.go` — transport route registration in task_06
- `internal/api/udsapi/routes.go` — transport parity wiring in task_07
- `web/src/systems/settings/types.ts` — frontend type wrappers and adapters in task_09

### Related ADRs

- [ADR-001: Use a consolidated settings namespace with a dedicated settings shell](adrs/adr-001.md) — Defines the API namespace and route-per-section model
- [ADR-003: Keep settings mutations restart-aware and separate from operational workflows](adrs/adr-003.md) — Defines restart action payloads and async status polling
- [ADR-004: Restrict HTTP settings mutations to loopback-bound servers in v1](adrs/adr-004.md) — Constrains mutation route exposure and extension parity

## Deliverables

- New settings and restart DTOs in `internal/api/contract`
- Updated OpenAPI surface for settings, restart actions, and required extension routes
- Regenerated API artifacts and web client types **(REQUIRED)**
- Contract and spec tests with >=80% coverage for the modified surface **(REQUIRED)**
- Validation that generated web types expose the fields needed by the settings UI **(REQUIRED)**

## Tests

- Unit tests:
  - [ ] Contract serialization covers `MutationResult`, restart action response, and restart status payloads
  - [ ] OpenAPI spec includes all required `/api/settings/*` routes and expected verbs
  - [ ] OpenAPI schemas include `write_target`, source-precedence metadata, and restart polling fields
  - [ ] Extension routes required by the settings UI are present in the HTTP-visible spec
- Integration tests:
  - [ ] Regenerated web types compile against the updated OpenAPI output
  - [ ] Spec validation or snapshot coverage catches route drift between contract changes and generated artifacts
- Test coverage target: >=80%
- All tests must pass

## Success Criteria

- All tests passing
- Test coverage >=80% for modified contract/spec packages
- HTTP, UDS, and `web/` share one authoritative settings API contract
- The web client can consume generated types for sections, collections, and restart status without ad hoc type holes
