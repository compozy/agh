---
status: completed
title: "Extension Model Source Contract"
type: backend
complexity: high
dependencies:
  - task_05
  - task_07
---

# Task 8: Extension Model Source Contract

## Overview
This task lets extensions contribute model source rows without owning catalog state. AGH remains the authority for validation, persistence, merge policy, and public projections.

<critical>
- ALWAYS READ `_techspec.md` and every ADR before starting
- REFERENCE TECHSPEC for implementation details - do not duplicate here
- FOCUS ON "WHAT" - describe what needs to be accomplished, not how
- MINIMIZE CODE - show code only to illustrate current structure or problem areas
- TESTS REQUIRED - every task MUST include tests in deliverables
</critical>

<requirements>
- MUST add manifest provide capability `model.source`.
- MUST add AGH-to-extension service method `models/list`.
- MUST add Host API methods `models/list`, `models/refresh`, and `models/status`.
- MUST capability-gate extension model source calls and Host API methods.
- MUST validate extension-returned rows before persistence or merge.
- MUST validate extension source IDs against the TechSpec `<kind>:<slug>` rule before persistence.
- MUST record extension source errors as source status and fail closed.
- MUST ensure marketplace/source-tier capability ceilings apply to model source grants.
</requirements>

## Subtasks
- [x] 8.1 Add protocol constants and manifest validation for `model.source` and `models/list`.
- [x] 8.2 Add extension source adapter that calls `models/list`, validates rows, and records source status.
- [x] 8.3 Add Host API model catalog list/refresh/status methods backed by daemon projections.
- [x] 8.4 Add capability checker mappings, grant tests, and marketplace ceiling tests.
- [x] 8.5 Add subprocess/extension integration tests for success, denied capability, malformed rows, and unavailable extension.

## Implementation Details
Follow `_techspec.md` section `Extension Protocol` and ADR-003. Extension-provided data is input to `internal/modelcatalog`; it is never the merged projection authority.

### Relevant Files
- `internal/extension/protocol/host_api.go` - capability and service method constants.
- `internal/extension/contract/host_api.go` - Host API method specs and payload contracts.
- `internal/extension/host_api.go` - Host API dispatch and handlers.
- `internal/extension/manager.go` - AGH-to-extension capability service calls.
- `internal/extension/capability.go` - capability checker mappings.
- `internal/extension/capability_test.go` - grant/denial tests.
- `.resources/paperclip/adapter-plugin.md` - adapter model endpoint reference.
- `.resources/paperclip/packages/adapters/acpx-local/src/server/execute.ts` - adapter execution and ACPX no-fake-session boundary reference.
- `/Users/pedronauck/dev/compozy/compozy-code/providers/sdk/src/models/types.ts` - model source contract reference.

### Dependent Files
- `sdk/typescript/src/generated/contracts.ts` - generated in Task 10.
- `openapi/agh.json` - generated in Task 10 if Host API docs expose payloads.
- `packages/site/content/runtime/` - extension author docs updated in Task 10.

### Related ADRs
- [ADR-003: Extension Model Source Contract](adrs/adr-003-extension-model-source-contract.md) - defines extension source authority and constraints.
- [ADR-001: Daemon-Owned Provider Model Catalog](adrs/adr-001-daemon-owned-provider-model-catalog.md) - daemon validates and merges extension rows.

### Web/Docs Impact
- `web/`: none directly unless generated extension/status types are consumed; Task 09 can display extension source status through catalog payloads.
- `packages/site`: Task 10 must document `model.source`, `models/list`, and Host API model methods.

### Extensibility / Agent Manageability / Config Lifecycle
- Extensibility: adds extension manifest capability, AGH-to-extension method, Host API methods, and capability checks.
- Agent manageability: Host API and native catalog surfaces let agents inspect extension-provided source status.
- Config lifecycle: no TOML keys added; extension source availability is governed by manifest grants and extension install state.

## Deliverables
- Extension capability and service method contract for model sources.
- Host API methods for catalog list/refresh/status.
- Capability checker and marketplace/source-tier grant behavior.
- Extension integration tests with 80%+ coverage for new methods **(REQUIRED)**.

## Tests
- Unit tests:
  - [x] manifest validation accepts `model.source` and rejects malformed capability declarations.
  - [x] capability checker maps Host API model methods to required grants.
  - [x] marketplace extension without grant cannot call model source methods.
  - [x] extension-returned malformed rows are rejected and recorded as source status.
  - [x] extension source names that cannot normalize to a valid source ID slug are rejected.
  - [x] Host API `models/list` returns daemon projection, not raw extension payload.
- Integration tests:
  - [x] extension subprocess implementing `models/list` contributes validated rows.
  - [x] denied extension source fails closed without blocking catalog list.
  - [x] unavailable extension records source error and preserves stale rows when present.
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- `go test ./internal/extension/...` passes.
- Extensions can provide model rows only through daemon-validated source contracts.

## Verification Evidence
- `go test ./internal/extension/... -count=1`: passed (`618 passed in 4 packages`).
- `go test ./internal/extension/... -race -count=1`: passed (`618 passed in 4 packages`).
- `go test ./internal/extension/... ./internal/modelcatalog ./internal/daemon -count=1`: passed (`1348 passed in 6 packages`).
- `go test ./internal/extension/... -coverprofile=.tmp/task08-extension-cover.out -covermode=atomic -count=1`: passed. New Task 08 surfaces are >=80% covered (`host_api_models.go` handlers 80.0-87.5%, `model_source.go` core functions 81.2-100.0%, `capability.go CheckHostAPI` 85.7%, `protocol/host_api.go CapabilityServiceMethods` 82.4%).
- AGH test convention helper passed for Task 08 new/conformed test files.
- `make codegen-check`: passed.
- `make lint`: passed (`0 issues`).
- `make verify`: passed before tracking completion.
