---
status: completed
title: "Expose provider metadata and provider_config through shared bridge APIs and OpenAPI"
type: backend
complexity: high
dependencies:
  - task_01
---

# Task 06: Expose provider metadata and provider_config through shared bridge APIs and OpenAPI

## Overview

The daemon API surface currently exposes only the old bridge shape and minimal provider metadata. This task updates shared bridge APIs, OpenAPI generation, and transport mappings so operators and the web UI can manage provider-specific configuration without abusing `delivery_defaults`.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
1. MUST expose `provider_config`, provider-declared secret-slot metadata, DM policy, and structured degradation data through the shared bridge API contracts.
2. MUST keep `delivery_defaults` narrowly scoped to delivery-target defaults and avoid mixing provider runtime configuration into that field.
3. MUST update HTTP, UDS, and OpenAPI surfaces consistently so generated web types and clients reflect the new bridge-management contract.
4. SHOULD preserve stable validation and error mapping for bridge CRUD operations while the payload shape expands.
</requirements>

## Subtasks

- [x] 6.1 Extend shared bridge request and response contracts for provider config and provider metadata
- [x] 6.2 Update bridge HTTP and UDS handlers to read and return the expanded payloads
- [x] 6.3 Regenerate and validate the OpenAPI surface for the new bridge contract
- [x] 6.4 Add API-level tests for CRUD, provider listing, and health payload changes

## Implementation Details

Follow the TechSpec sections "Data Model Changes", "Provider Manifest", and "Impact Analysis". This task should stop at shared APIs and code-generated schema updates; it should not implement the web UI or provider binaries.

### Relevant Files

- `internal/api/contract/bridges.go` — Shared bridge payloads currently expose only `delivery_defaults`
- `internal/api/core/bridges.go` — Core bridge handlers must return the expanded bridge and provider metadata
- `internal/api/httpapi/bridges_test.go` — HTTP bridge contract coverage should grow with the new payload shape
- `internal/api/spec/spec.go` — OpenAPI generation must reflect the updated bridge management schema

### Dependent Files

- `web/src/generated/agh-openapi.d.ts` — Generated client types later drive the bridge UI changes
- `web/src/systems/bridges/types.ts` — Web-facing bridge models later depend on the new schema
- `internal/api/udsapi/bridges_test.go` — UDS bridge transport tests need updated expectations

### Reference Sources (.resources/)

- `.resources/chat/packages/chat/src/types.ts` — Chat-SDK adapter config properties (`name`, `userName`, `botUserId`, `lockScope`, `persistMessageHistory`); reference for what provider metadata operators need to see
- `.resources/hermes/gateway/platforms/ADDING_A_PLATFORM.md` — Hermes 16-item integration checklist for adding a platform; shows the operator-facing metadata surface a mature multi-adapter system exposes

### Related ADRs

- [ADR-001: Provider-Scoped Bridge SDK and Runtime Model](adrs/adr-001.md) — Explains why provider metadata must be surfaced independently from process ownership
- [ADR-003: Bridge V1 Scope Instead of Full Chat-SDK Parity](adrs/adr-003.md) — Establishes provider configuration and DM policy as part of v1

## Deliverables

- Updated shared bridge API contracts for provider config, provider metadata, and degradation reporting
- HTTP, UDS, and OpenAPI surfaces aligned to the expanded bridge-management schema
- Regenerated typed schema outputs consumed by the web client
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration tests for bridge API contract behavior **(REQUIRED)**

## Tests

- Unit tests:
  - [x] create and update bridge payload mapping keeps `provider_config` distinct from `delivery_defaults`
  - [x] provider-listing payloads include declared secret slots and optional config schema hints when present
  - [x] bridge health payload mapping includes structured degradation fields without breaking existing counters
- Integration tests:
  - [x] POST and GET bridge APIs round-trip `provider_config` and delivery defaults independently
  - [x] bridge provider listing surfaces provider metadata needed by operators and the web client
  - [x] generated OpenAPI schema includes the new bridge fields and no longer treats provider config as unknown-only state
- Test coverage target: >=80%
- All tests must pass

## Success Criteria

- All tests passing
- Test coverage >=80%
- Operators can configure provider-owned runtime settings through stable shared APIs
- OpenAPI and transport layers expose the bridge-management contract needed by the UI
