---
status: completed
title: Add Shared API Contracts and Codegen Surface
type: backend
complexity: high
dependencies:
  - task_04
  - task_09
---

# Task 10: Add Shared API Contracts and Codegen Surface

## Overview

Define the public contract types for Soul, Heartbeat, session health, wake audit/status, authoring mutations, and read models. This task ensures HTTP, UDS, CLI, web, SDK, tools, and extensions share one generated contract surface before transports are wired.

<critical>
- ALWAYS READ `_techspec.md`, `_techspec_soul.md`, `_techspec_heartbeat.md`, every ADR, and current contract/codegen instructions before editing contracts.
- REFERENCE TECHSPEC for DTO names, redaction, CAS fields, route payloads, and generated surfaces.
- FOCUS ON WHAT must be contractually stable: request/response types, enums, diagnostics, provenance, and generated consumers.
- MINIMIZE CODE in task notes; do not implement route logic here beyond contract plumbing required for codegen.
- TESTS REQUIRED for schema generation, enum closure, redaction, CAS body shape, and generated TypeScript compatibility.
- NO WORKAROUNDS: do not expose raw prompt bodies, raw tokens, HTTP-only behavior, or divergent UDS/HTTP DTOs.
</critical>

<requirements>
- MUST activate `agh-contract-codegen-coship`, `agh-code-guidelines`, and `golang-pro`.
- MUST activate `agh-test-conventions`, `testing-anti-patterns`, and `typescript-advanced` when touching generated TypeScript consumers/tests.
- MUST add contract DTOs for Soul read models, Soul authoring, Heartbeat policy/status, Heartbeat authoring, session health, wake state, and wake events.
- MUST represent mutation CAS as body-level `expected_digest`.
- MUST define closed enums for health states, wake reasons, diagnostics severity, and validation status.
- MUST apply redaction rules so raw secrets, raw provider tokens, claim tokens, and disallowed prompt data are not serialized.
- MUST update OpenAPI generation and generated TypeScript/sdk contract outputs in the same change.
- MUST run `make codegen` and `make codegen-check`.
</requirements>

## Subtasks
- [x] 10.1 Add Go contract DTOs and closed enums for Soul, Heartbeat, session health, and wake audit.
- [x] 10.2 Add OpenAPI annotations/spec registration for the new DTOs and payloads.
- [x] 10.3 Add redaction and conversion tests for contract payloads.
- [x] 10.4 Run codegen and commit generated OpenAPI, web, and SDK type outputs.
- [x] 10.5 Add TypeScript contract smoke tests where generated consumers already have coverage.
- [x] 10.6 Verify HTTP and UDS can share these DTOs without transport-specific divergence.

## Implementation Details

Keep contract types under the existing `internal/api/contract` and OpenAPI generation pattern. Route implementation comes in task_11; this task should make the schema and generated clients ready.

### Relevant Files
- `internal/api/contract/agents.go` - likely existing agent DTO home.
- `internal/api/contract/responses.go` - shared response and diagnostic shapes.
- `internal/api/contract/` - destination for new Soul, Heartbeat, session health, and wake DTOs.
- `internal/api/spec/spec.go` - OpenAPI schema registration.
- `cmd/agh-codegen/` - codegen entrypoint if schema registration changes.
- `openapi/agh.json` - generated OpenAPI output.
- `web/src/generated/agh-openapi.d.ts` - generated web TypeScript contract.
- `sdk/typescript/src/generated/contracts.ts` - generated SDK contract output.

### Dependent Files
- `internal/api/contract/*_test.go` - DTO serialization, redaction, enum, and schema tests.
- `web/src/generated/agh-openapi.d.ts` - generated output consumed by task_14.
- `sdk/typescript/src/generated/contracts.ts` - generated output consumed by task_13 and task_14.
- `.compozy/tasks/agent-soul/task_11.md` - implements routes using these contracts.
- `.compozy/tasks/agent-soul/task_13.md` - exposes extension/SDK surfaces using these contracts.
- `.compozy/tasks/agent-soul/task_14.md` - updates web and SDK consumers after codegen.

### Related ADRs
- [ADR-002: Soul Prompt and Read Model Exposure](adrs/adr-002.md) - requires read-model and context projection surfaces.
- [ADR-006: Managed Soul Authoring in v1](adrs/adr-006.md) - requires mutation DTOs and CAS.
- [ADR-010: Managed Heartbeat and Session Health Surfaces](adrs/adr-010.md) - requires route/transport parity.
- [ADR-011: Config Authority for Cadence and Wake Limits](adrs/adr-011.md) - requires effective config reporting.

## Extensibility / Agent Manageability / Config Lifecycle
- Extensibility: generated contracts are the source for Host API, TypeScript SDK, tools/resources, bridge SDKs, and future public docs.
- Agent manageability: DTOs must support CLI JSON, HTTP responses, UDS responses, deterministic errors, and status/config discovery.
- Config lifecycle: expose effective config digest/provenance for Soul and Heartbeat; do not add new config keys.

### Web/Docs Impact
- Web impact: generated `web/src/generated/agh-openapi.d.ts` changes are required; task_14 updates consumer tests.
- Docs impact: OpenAPI and CLI docs in task_15 must describe final payloads and redaction behavior.

## Deliverables
- Go contract DTOs and OpenAPI schema registration.
- Closed enums and redacted diagnostic/authoring/status payloads.
- Generated `openapi/agh.json`, web TypeScript contract, and TypeScript SDK contract outputs.
- Contract and generated-consumer tests.
- `make codegen` and `make codegen-check` evidence.

## Tests
- Unit tests:
  - [x] DTO serialization includes required fields and omits forbidden raw data.
  - [x] Closed enums reject or fail conversion for unknown health/wake/diagnostic states.
  - [x] CAS mutation payloads require `expected_digest` in request bodies.
  - [x] Config provenance and digest fields serialize deterministically.
  - [x] OpenAPI schema contains all new request and response types.
- Integration tests:
  - [x] `make codegen-check` passes after generated files are updated.
  - [x] Generated TypeScript contracts compile in the workspace typecheck lane.
- Test coverage target: >=80%.
- All tests must pass.

## References
- `_techspec.md` - aggregate contract and surface parity requirements.
- `_techspec_soul.md` - Soul contract requirements.
- `_techspec_heartbeat.md` - Heartbeat/session health contract requirements.
- `.compozy/tasks/agent-soul/analysis/analysis_openclaw_heartbeat.md` - gateway/protocol shape precedent.
- `.resources/openclaw/docs/gateway/protocol.md:313-438` - protocol payload precedent.
- `.resources/openclaw/src/gateway/protocol/schema/agent.ts:131-213` - typed agent schema precedent.

## Success Criteria
- All tests passing.
- Test coverage >=80%.
- HTTP, UDS, CLI, web, SDK, extensions, tools, and docs can share one contract surface.
- Generated contracts are current and `make codegen-check` passes.
