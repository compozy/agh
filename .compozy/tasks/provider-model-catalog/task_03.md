---
status: pending
title: "Model Catalog Service and Catalog Sources"
type: backend
complexity: critical
dependencies:
  - task_01
  - task_02
---

# Task 3: Model Catalog Service and Catalog Sources

## Overview
This task creates the `internal/modelcatalog` service boundary and the non-live sources that are available without provider network calls. It turns provider config, builtin defaults, and `models.dev` catalog data into a deterministic merged projection with stale fallback and source status.

<critical>
- ALWAYS READ `_techspec.md` and every ADR before starting
- REFERENCE TECHSPEC for implementation details - do not duplicate here
- FOCUS ON "WHAT" - describe what needs to be accomplished, not how
- MINIMIZE CODE - show code only to illustrate current structure or problem areas
- TESTS REQUIRED - every task MUST include tests in deliverables
</critical>

<requirements>
- MUST add `internal/modelcatalog` with the service, source, store, row, model, source status, and reasoning effort concepts from the TechSpec.
- MUST implement deterministic merge by `(provider_id, model_id)`, source priority, row freshness, and source ID tie-break.
- MUST implement merged availability states (`available_live`, `available_stale`, `unavailable_live`, `unavailable_stale`, `unknown`) exactly as specified.
- MUST treat lower-priority sources as enrichment only when higher-priority fields are absent.
- MUST support partial-source success, all-source failure, source status, stale fallback, TTL checks, and redacted errors.
- MUST implement `builtin`, `config`, and `models_dev` sources.
- MUST parse both current and legacy `models.dev` field names listed in the TechSpec.
- MUST honor `[model_catalog.sources.models_dev]` enabled/endpoint/TTL/timeout config and never fetch when disabled.
- MUST validate dynamic `source_id` values using the TechSpec slug rule.
- MUST use explicit HTTP timeouts and never store raw upstream payloads or secrets.
</requirements>

## Subtasks
- [ ] 3.1 Create the `internal/modelcatalog` package and public service/source/store types.
- [ ] 3.2 Implement priority/freshness/source-ID merge, field enrichment, merged availability, source status, stale, and partial failure semantics.
- [ ] 3.3 Implement builtin and config sources from the new provider model config shape.
- [ ] 3.4 Implement `models.dev` fetch/parse/cache/config behavior with current and legacy field aliases.
- [ ] 3.5 Add redaction, timeout, and deterministic sorting behavior.
- [ ] 3.6 Add focused package tests for merge, sources, parser variants, stale fallback, and failure modes.
- [ ] 3.7 Add source ID slug validation and deterministic source ordering tests.

## Implementation Details
Follow `_techspec.md` sections `Core Interfaces`, `Source Implementations`, and `Safety Invariants`. Use `compozy-code` as a reference for source prioritization and merge behavior, but keep AGH-specific provider mapping.

### Relevant Files
- `internal/modelcatalog/` - new package to create.
- `internal/config/provider.go` - source data for builtin/config sources after Task 01.
- `internal/store/globaldb/` - store implementation from Task 02.
- `/Users/pedronauck/dev/compozy/compozy-code/providers/sdk/src/models/model-discovery-service.ts` - reference for parallel source querying, partial success, and priority merge.
- `/Users/pedronauck/dev/compozy/compozy-code/providers/sdk/src/models/catalog-sources/models-dev-source.ts` - reference for `models.dev` source and cache behavior.
- `/Users/pedronauck/dev/compozy/compozy-code/providers/sdk/src/models/types.ts` - reference source interface shape.

### Dependent Files
- `internal/daemon/` - Task 05 wires the service into runtime.
- `internal/api/core/handlers.go` - Task 07 injects service into API handlers.
- `internal/extension/` - Task 08 adds extension source adapters.

### Related ADRs
- [ADR-001: Daemon-Owned Provider Model Catalog](adrs/adr-001-daemon-owned-provider-model-catalog.md) - defines daemon-owned catalog service authority.
- [ADR-002: Provider Model Config Hard Cut](adrs/adr-002-provider-model-config-hard-cut.md) - config source depends on new model config shape.

### Web/Docs Impact
- `web/`: none directly in this task - checked service package only; web consumes generated API from later tasks.
- `packages/site`: none directly in this task - behavior is documented after public surfaces exist.

### Extensibility / Agent Manageability / Config Lifecycle
- Extensibility: source interface becomes the internal adapter point used by extension sources in Task 08.
- Agent manageability: no direct surface yet; service methods must support later CLI/HTTP/UDS list/refresh/status.
- Config lifecycle: config source reads `providers.<id>.models.default` and `models.curated` only.

## Deliverables
- `internal/modelcatalog` service and source/store contracts.
- Builtin, config, and `models.dev` sources with AGH-specific provider mapping and source config support.
- Redacted source error handling and deterministic projection sorting.
- Unit tests with 80%+ coverage for service/source behavior **(REQUIRED)**.
- `httptest` coverage for `models.dev` parser/cache/fallback behavior **(REQUIRED)**.

## Tests
- Unit tests:
  - [ ] higher-priority source wins conflicting fields.
  - [ ] provider live priority wins over extension priority when both provide conflicting live metadata.
  - [ ] equal-priority/equal-freshness conflicts resolve by ascending `source_id`.
  - [ ] lower-priority source fills missing context/cost/display metadata.
  - [ ] stale live `available=true` projects as `available_stale`, not `unknown`.
  - [ ] stale live `available=false` projects as `unavailable_stale`, not `unknown`.
  - [ ] catalog-only `models.dev` rows project `availability_state=unknown` and `available=null`.
  - [ ] partial success returns merged results and records failed source status.
  - [ ] all-source failure returns an error when no stale rows exist.
  - [ ] stale rows are returned and labeled when refresh fails after prior success.
  - [ ] source errors redact secret-shaped values.
  - [ ] `models.dev` parser accepts `reasoning`, `tool_call`, `limit.context`, `limit.input`, `limit.output`, and `cost`.
  - [ ] `models.dev` parser accepts legacy `supportsReasoning`, `supports_tools`, `contextWindow`, and pricing aliases.
  - [ ] disabled `models.dev` source records disabled status and performs no outbound request.
  - [ ] overridden `models.dev` endpoint, TTL, and timeout are applied.
  - [ ] invalid extension source ID slug is rejected before persistence.
- Integration tests:
  - [ ] service backed by global DB store can refresh then list rows by provider.
  - [ ] config source accepts manual default outside curated list.
  - [ ] no raw upstream `models.dev` payload is persisted.
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- `go test ./internal/modelcatalog ./internal/modelcatalog/...` passes.
- Service behavior matches TechSpec safety invariants for stale fallback, partial success, and redaction.
