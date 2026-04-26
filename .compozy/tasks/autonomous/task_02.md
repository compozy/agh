---
status: pending
title: Agent Contract DTOs And OpenAPI Parity
type: backend
complexity: high
dependencies:
  - task_01
---

# Task 02: Agent Contract DTOs And OpenAPI Parity

## Overview
Define the transport contracts needed by agent-facing autonomy APIs before behavior depends on them. This task adds DTOs for agent context, claim/lease responses, spawn metadata, coordinator config read models, and safe HTTP projections while keeping generated OpenAPI and web TypeScript in lockstep.

<critical>
- ALWAYS READ `_techspec.md`, ADR-002, ADR-003, ADR-006, and ADR-011 before changing contracts
- REFERENCE TECHSPEC for payload shape; do not expose raw `claim_token` in read models
- FOCUS ON "WHAT" - stable DTOs, OpenAPI parity, and generated web types
- MINIMIZE CODE - do not implement business behavior here
- TESTS REQUIRED - OpenAPI spec, generated types, and contract conversion tests must move together
- NO WORKAROUNDS - do not use `any`, loose maps, or non-null assertions in web type consumers to silence contract drift
</critical>

<requirements>
- MUST add transport-agnostic DTOs under `internal/api/contract` for agent context, task claim/lease commands, spawn, lineage, and coordinator config.
- MUST keep raw `claim_token` limited to the synchronous claim response; read/list/detail models MUST expose only `claim_token_hash` when needed.
- MUST update OpenAPI spec registration and tests for every new public endpoint or schema.
- MUST run `make codegen` and update `openapi/agh.json` and `web/src/generated/agh-openapi.d.ts`.
- MUST update affected `web/src/systems/*/types.ts`, Storybook/MSW fixtures, and API contract tests if generated DTOs affect existing consumers.
- MUST not add new web pages, scheduler dashboards, or coordinator UI routes.
</requirements>

## Subtasks
- [ ] 2.1 Add contract DTOs for `/agent/me`, `/agent/context`, task lease commands, spawn, lineage, and coordinator config read surfaces.
- [ ] 2.2 Add OpenAPI operation specs/tests for the new schemas without wiring runtime handlers yet.
- [ ] 2.3 Regenerate OpenAPI and generated TypeScript contracts.
- [ ] 2.4 Update web type derivations and fixtures affected by task-run/session-lineage fields.
- [ ] 2.5 Add contract tests proving raw claim tokens never appear in read models, SSE payloads, or web-facing DTOs.
- [ ] 2.6 Run backend contract tests plus `make codegen`, `make web-typecheck`, and `make web-test`.

## Implementation Details
Use `internal/api/contract` as the source of truth and the existing `internal/api/spec` builder for OpenAPI. Any field added for task runs or sessions must flow through conversion helpers rather than ad-hoc response structs in handlers.

### Relevant Files
- `internal/api/contract/contract.go` - session and shared payloads.
- `internal/api/contract/tasks.go` - task-run payloads and read models.
- `internal/api/spec/spec.go` - OpenAPI operation/schema registration.
- `internal/api/spec/spec_test.go` - OpenAPI schema and endpoint assertions.
- `web/src/generated/agh-openapi.d.ts` - generated TypeScript contract output.
- `web/src/lib/api-contract.ts` - generated operation typing consumed by systems.
- `web/src/systems/tasks/types.ts` - task/run type derivations.
- `web/src/systems/session/types.ts` - session type derivations.
- `.resources/multica/packages/core/api/client.ts` - reference for typed API client boundaries.
- `.resources/paperclip/cli/src/client/http.ts` - reference for CLI/API transport shaping.
- `.resources/claude-code/Tool.ts` - reference for typed tool/command metadata.

### Dependent Files
- `internal/api/httpapi/*` and `internal/api/udsapi/*` - later handler tasks consume these DTOs.
- `internal/cli/client.go` - later CLI methods alias or consume these DTOs.
- `web/src/systems/tasks/mocks/fixtures.ts` - generated contract updates can affect fixtures.
- `web/src/systems/session/mocks/fixtures.ts` - lineage fields can affect session fixtures.

### Related ADRs
- [ADR-002: Agent-Facing CLI Before Built-In MCP Tools](adrs/adr-002.md) - contracts back the CLI-first surface.
- [ADR-003: Extend Task Runs for Atomic Claim and Lease](adrs/adr-003.md) - claim token exposure rules.
- [ADR-006: Safe Spawn Requires Lineage, TTL, and Permission Narrowing](adrs/adr-006.md) - spawn/lineage DTOs.
- [ADR-011: Generated Contracts and Documentation Co-Ship with Autonomy MVP Steps](adrs/adr-011.md) - codegen and web parity.

## Deliverables
- Contract DTOs and OpenAPI specs for autonomy MVP surfaces.
- Regenerated `openapi/agh.json` and `web/src/generated/agh-openapi.d.ts`.
- Updated web contract consumers/fixtures where generated fields require it.
- Unit tests with 80%+ coverage for contract conversion helpers **(REQUIRED)**.
- OpenAPI/codegen and web type/test verification **(REQUIRED)**.

## Tests
- Unit tests:
  - [ ] Contract conversion includes lineage, task lease summaries, coordinator config, and bounded context sections.
  - [ ] Read models include `claim_token_hash` only and never include raw `claim_token`.
  - [ ] Empty optional sections serialize consistently without `null`/missing-field ambiguity that breaks generated clients.
  - [ ] OpenAPI schema tests assert required fields and operation tags for agent endpoints.
  - [ ] Web type derivations compile against regenerated task/session payloads without assertions or loose `any`.
- Integration tests:
  - [ ] `make codegen` produces no stale generated artifacts.
  - [ ] `make web-typecheck` and `make web-test` pass after generated contract changes.
- Test coverage target: >=80%.
- All tests must pass.

## Success Criteria
- All tests passing.
- Test coverage >=80%.
- Generated contracts are current and safe for later API/CLI tasks.
- No raw claim token is exposed outside the issuing response path.
