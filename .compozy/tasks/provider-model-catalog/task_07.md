---
status: completed
title: "HTTP, UDS, CLI, and OpenAI Model Projection Surfaces"
type: backend
complexity: critical
dependencies:
  - task_05
---

# Task 7: HTTP, UDS, CLI, and OpenAI Model Projection Surfaces

## Overview
This task exposes the daemon-owned model catalog through AGH's native operator and agent surfaces. It adds shared contract payloads, HTTP/UDS parity, CLI commands, and the HTTP-only OpenAI-compatible `/api/openai/v1/models` projection.

<critical>
- ALWAYS READ `_techspec.md` and every ADR before starting
- REFERENCE TECHSPEC for implementation details - do not duplicate here
- FOCUS ON "WHAT" - describe what needs to be accomplished, not how
- MINIMIZE CODE - show code only to illustrate current structure or problem areas
- TESTS REQUIRED - every task MUST include tests in deliverables
</critical>

<requirements>
- MUST add native catalog endpoints for list, provider list, refresh, provider refresh, status, and provider status.
- MUST implement HTTP and UDS routes through shared `internal/api/core` handlers.
- MUST add HTTP-only OpenAI-compatible `GET /api/openai/v1/models` using `agh` metadata for AGH-specific fields.
- MUST apply the same HTTP auth/middleware contract as `/api/*`, return OpenAI-shaped errors for unauthorized requests, and avoid UDS registration for this route.
- MUST add `agh provider models list|refresh|status` with structured `-o json` output.
- MUST expose stale/source/error/status fields needed by agents.
- MUST produce deterministic validation and service-unavailable errors.
- MUST co-ship contract source, `openapi/agh.json`, `web/src/generated/agh-openapi.d.ts`, contract tests, `make codegen`, and `make codegen-check` for the new routes.
</requirements>

## Subtasks
- [x] 7.1 Add contract payloads and conversion helpers for catalog models, source refs, status, refresh results, availability state, and `/api/openai/v1/models`.
- [x] 7.2 Add shared `BaseHandlers` methods for catalog list/refresh/status.
- [x] 7.3 Register HTTP and UDS routes with route tests.
- [x] 7.4 Add OpenAI-compatible `/api/openai/v1/models` route, auth/error, HTTP-only, and projection tests.
- [x] 7.5 Add `agh provider models list|refresh|status` commands with JSON output.
- [x] 7.6 Add cross-surface tests comparing CLI, HTTP, and UDS results for the same catalog state, including canonical JSON byte equality for one native payload.

## Implementation Details
Follow `_techspec.md` sections `Public Interfaces`, `Agent Manageability Plan`, and `Safety Invariants`. Activate `agh-contract-codegen-coship`, `agh-code-guidelines`, `golang-pro`, `agh-test-conventions`, and `testing-anti-patterns`.

### Relevant Files
- `internal/api/contract/contract.go` - shared catalog payloads and `/api/openai/v1/models` payloads.
- `internal/api/core/handlers.go` - handler config and dependency accessors.
- `internal/api/core/` - new model catalog handler file and tests.
- `internal/api/httpapi/routes.go` - HTTP route registration.
- `internal/api/udsapi/routes.go` - UDS route registration.
- `internal/cli/provider.go` - existing provider CLI namespace.
- `internal/cli/config.go` - structured output command patterns.
- `openapi/agh.json` - regenerate in this task for new API contract payloads/routes.
- `web/src/generated/agh-openapi.d.ts` - regenerate in this task for new generated web types.
- `/Users/pedronauck/dev/compozy/compozy-code/providers/sdk/src/server/routes-models.ts` - OpenAI-compatible model route reference.
- `.resources/paperclip/adapter-plugin.md` - adapter-facing model endpoint shape reference.
- `.resources/paperclip/packages/adapters/opencode-local/src/server/models.ts` - model list route and validation behavior reference.
- `.resources/harnss/src/types/window.d.ts` - IPC model/config surface reference.

### Dependent Files
- `sdk/typescript/src/generated/contracts.ts` - regenerated in Task 10 if extension/SDK docs expose additional model contracts.
- `packages/site/content/runtime/cli-reference/` - regenerated in Task 10.
- `internal/extension/contract/host_api.go` - Task 08 adds extension-specific model methods.

### Related ADRs
- [ADR-001: Daemon-Owned Provider Model Catalog](adrs/adr-001-daemon-owned-provider-model-catalog.md) - requires HTTP/UDS/CLI/web/OpenAI projection.

### Web/Docs Impact
- `web/`: Task 09 consumes the new generated contract and endpoints under a model catalog system.
- `packages/site`: Task 10 documents endpoints, `/api/openai/v1/models`, and CLI commands; `make cli-docs` must regenerate CLI reference.

### Extensibility / Agent Manageability / Config Lifecycle
- Extensibility: provides daemon catalog surfaces used by extension Host API in Task 08.
- Agent manageability: adds CLI, HTTP, and UDS structured list/refresh/status paths.
- Config lifecycle: refresh/list/status responses must reflect config-derived source rows from `providers.<id>.models`.

## Deliverables
- Native HTTP and UDS model catalog routes.
- HTTP-only OpenAI-compatible `GET /api/openai/v1/models` route with `agh` metadata and OpenAI-shaped errors.
- `agh provider models list|refresh|status` commands.
- Contract, route, CLI, and parity tests with 80%+ coverage **(REQUIRED)**.
- Regenerated OpenAPI and web TypeScript contract outputs for this route batch.

## Tests
- Unit tests:
  - [x] contract payload conversion preserves nullable availability and stale/source fields.
  - [x] `/api/openai/v1/models` uses `agh` metadata and supports `provider_id` filter.
  - [x] `/api/openai/v1/models` requires the same bearer auth as `/api/*` and returns OpenAI-shaped unauthorized errors.
  - [x] `/api/openai/v1/models` is not registered on UDS routes.
  - [x] invalid provider ID returns deterministic error.
  - [x] refresh service failure returns source status when available.
  - [x] CLI `list` prints structured JSON with model/source/status fields.
  - [x] CLI `refresh` returns source statuses, not only success text.
- Integration tests:
  - [x] HTTP and UDS list routes return matching payloads for the same seeded catalog.
  - [x] HTTP and UDS native list route canonical JSON bytes match for a deterministic seeded payload.
  - [x] CLI output matches HTTP/UDS state for list/status.
  - [x] service unavailable paths are consistent across HTTP/UDS/CLI.
  - [x] `make codegen` and `make codegen-check` pass with no drift after route/contract additions.
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- `go test ./internal/api/... ./internal/cli` passes.
- Agents can list, refresh, and inspect provider model catalog state without web UI.
