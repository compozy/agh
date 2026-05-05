---
status: completed
title: Explicit Session Provider Contracts and Generated Surfaces
type: backend
complexity: high
dependencies:
  - task_02
---

# Task 04: Explicit Session Provider Contracts and Generated Surfaces

## Overview

Expose the session provider override on every explicit creation and read surface that an operator or extension can call directly. This task updates the shared contracts, HTTP/UDS handlers, CLI, extension Host API, and generated artifacts so `provider` becomes part of the public session API instead of remaining an internal-only field.

<critical>
- ALWAYS READ `_techspec.md`, ADRs, and task_02 before starting (`_prd.md` is absent for this feature)
- REFERENCE TECHSPEC sections "API Endpoints", "Testing Approach", and "Technical Dependencies"
- CONTRACTS MUST STAY SHARED - HTTP, UDS, CLI, extension host API, and generated TS types must agree on the same `provider` shape
- CODEGEN IS PART OF THE TASK - do not leave `openapi/agh.json` or `web/src/generated/agh-openapi.d.ts` stale
- CLI AND HOST API ARE FIRST-CLASS EXPLICIT SURFACES - they must accept provider override and show effective provider in responses/output
- GREENFIELD: nao aceitar contracts parcialmente atualizados ou payloads divergentes entre transportes
</critical>

<requirements>
- MUST add optional `provider` to `contract.CreateSessionRequest`
- MUST add `provider` to `contract.SessionPayload`
- MUST update HTTP and UDS session create/read handlers to decode, forward, and emit the provider field
- MUST add `--provider` to `agh session new`
- MUST surface the effective provider in CLI session list/detail output
- MUST extend extension Host API `sessions.create` to accept optional `provider`
- MUST regenerate `openapi/agh.json` and `web/src/generated/agh-openapi.d.ts`
- MUST include `make codegen-check` in the verification flow for this task
</requirements>

## Subtasks
- [x] 4.1 Extend shared session request/response contracts with `provider`
- [x] 4.2 Update HTTP and UDS handlers plus payload conversions to round-trip provider
- [x] 4.3 Add CLI support for `--provider` and provider-aware session output
- [x] 4.4 Extend the extension Host API `sessions.create` contract with optional provider
- [x] 4.5 Regenerate OpenAPI and checked-in web types, then add transport-level coverage

## Implementation Details

See TechSpec "API Endpoints", "Testing Approach", and ADR-004. This task should not invent new endpoints or side channels. It should extend the explicit session creation/read surfaces that already exist and keep all generated artifacts in sync.

### Relevant Files
- `internal/api/contract/contract.go` - shared request/response structs for HTTP, UDS, CLI, and generated schema
- `internal/api/contract/contract_test.go` - request decoding and payload shape coverage
- `internal/api/core/handlers.go` - shared handler wiring and session payload envelopes
- `internal/api/core/session_workspace.go` - create/resume handlers that pass `CreateOpts`
- `internal/api/core/conversions.go` - `session.Info` to `SessionPayload` conversion
- `internal/api/spec/spec.go` - OpenAPI schema wiring
- `internal/cli/session.go` - CLI flags and output formatting
- `internal/cli/session_test.go` - CLI contract and output coverage
- `internal/extension/host_api.go` - extension create-session surface
- `internal/extension/protocol/host_api.go` - host API protocol types
- `internal/extension/host_api_test.go` - host API behavior coverage
- `internal/extension/host_api_integration_test.go` - integration path coverage
- `cmd/agh-codegen/main.go` - codegen entrypoint
- `openapi/agh.json` - checked-in generated OpenAPI artifact
- `web/src/generated/agh-openapi.d.ts` - checked-in generated web types

### Dependent Files
- `web/src/systems/session/adapters/session-api.ts` - task_06 consumes the new generated session create/read shapes
- `web/src/systems/session/types.ts` - task_06 aligns client-facing types to generated provider fields
- `.compozy/tasks/session-driver-override/task_06.md` - depends on these explicit contracts and generated artifacts
- `.compozy/tasks/session-driver-override/task_08.md` - QA execution must verify CLI/API parity on the provider field

### Related ADRs
- [ADR-003: Persist Effective Session Provider And Fail Explicitly On Mismatch](adrs/adr-003.md) - transport payloads must expose the persisted provider clearly
- [ADR-004: Use Explicit Session Creation Surfaces For Provider Selection](adrs/adr-004.md) - defines which surfaces expose provider choice

## Deliverables
- Shared create/read contracts extended with `provider`
- HTTP/UDS handlers updated to accept and emit provider
- CLI support for `agh session new --provider` plus provider-aware list/detail output
- Extension Host API `sessions.create` updated for optional provider override
- Regenerated `openapi/agh.json` and `web/src/generated/agh-openapi.d.ts` **(REQUIRED)**
- Transport and codegen coverage proving contract parity **(REQUIRED)**

## Tests
- Unit tests:
  - [x] Request decoding accepts optional `provider` without breaking requests that omit it
  - [x] Session payload conversion emits the effective provider from `session.Info`
  - [x] CLI create command forwards `--provider` correctly
  - [x] CLI session list/detail output includes the effective provider
  - [x] Extension Host API `sessions.create` accepts and forwards provider
- Integration tests:
  - [x] HTTP create with explicit provider returns session payloads containing the same effective provider
  - [x] UDS create/read flows expose provider consistently with HTTP
  - [x] `make codegen-check` passes after updating generated artifacts
  - [x] Generated TS types reflect the new session and workspace payload fields used by the web client
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- Every explicit session surface can create and read sessions with provider state coherently
- Generated artifacts stay in lockstep with the shared contracts
