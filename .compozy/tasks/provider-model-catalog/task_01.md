---
status: completed
title: "Provider Config and Builtin Model Hard Cut"
type: backend
complexity: critical
dependencies: []
---

# Task 1: Provider Config and Builtin Model Hard Cut

## Overview
This task replaces the old flat provider model fields with the nested `models` block from the TechSpec. It is the hard-cut foundation and contract/codegen co-ship boundary for every later catalog, ACP, API, settings, and web task.

<critical>
- ALWAYS READ `_techspec.md` and every ADR before starting
- REFERENCE TECHSPEC for implementation details - do not duplicate here
- FOCUS ON "WHAT" - describe what needs to be accomplished, not how
- MINIMIZE CODE - show code only to illustrate current structure or problem areas
- TESTS REQUIRED - every task MUST include tests in deliverables
</critical>

<requirements>
- MUST remove provider-level `default_model`, `supported_models`, and `supports_reasoning_effort` from config structs, merge overlays, settings models, payload conversions, and tests.
- MUST introduce the nested `ProviderModelsConfig` and curated model metadata shape described in the TechSpec.
- MUST introduce model catalog source config and provider discovery config from the TechSpec.
- MUST update builtin providers so defaults and curated model metadata live under the new `models` block.
- MUST reject old TOML keys with deterministic hard-cut errors and no aliases, dual reads, or compatibility fallback paths.
- MUST preserve the rule that manual model IDs remain valid even when absent from `models.curated`.
- MUST update settings/config/bootstrap/tool-surface paths that currently read or write the old flat fields.
- MUST co-ship API contract removals, OpenAPI regeneration, generated TypeScript updates, and web/settings consumers needed for `make verify` to pass without old-field residue.
</requirements>

## Subtasks
- [x] 1.1 Replace old provider model fields with the nested provider models config shape, model catalog source config, and provider discovery config.
- [x] 1.2 Move builtin provider defaults and curated model suggestions into the new config shape.
- [x] 1.3 Remove old effective helper functions and validation for `supported_models` / `supports_reasoning_effort`.
- [x] 1.4 Update config merge, clone, persistence, bootstrap, CLI config, tool-surface, settings, API contract, conversion, generated OpenAPI/TypeScript, minimal web settings consumers, and workspace clone paths.
- [x] 1.5 Add hard-cut validation coverage for old keys and full parse/merge/render coverage for the new shape.
- [x] 1.6 Update backend fixtures and tests that still assert old provider field names.
- [x] 1.7 Run `make codegen`, `make codegen-check`, `make bun-typecheck`, and focused web settings tests required by the contract hard cut.

## Implementation Details
Follow `_techspec.md` sections `Delete Targets`, `Core Interfaces`, and `Config Lifecycle`. Activate `agh-code-guidelines`, `golang-pro`, `agh-test-conventions`, and `testing-anti-patterns` before editing Go code/tests.

### Relevant Files
- `internal/config/provider.go` - current provider config fields, builtin providers, helpers, validation, clone logic.
- `internal/config/merge.go` - current provider overlay fields for old TOML keys.
- `internal/config/persistence.go` - config rendering and persistence behavior.
- `internal/config/bootstrap.go` - bootstrap config generation currently sensitive to model defaults.
- `internal/config/tool_surface.go` - agent-facing config/tool surface key exposure.
- `internal/cli/config.go` - CLI config mutation paths for provider settings.
- `internal/settings/models.go` - editable provider settings model.
- `internal/settings/collections.go` - settings collection projection and update paths.
- `internal/workspace/clone.go` - provider config deep-copy behavior.
- `internal/api/contract/settings.go` - provider settings payload old field shape.
- `internal/api/core/conversions.go` - settings/session provider payload conversion.
- `openapi/agh.json` - regenerate immediately after contract removal.
- `web/src/generated/agh-openapi.d.ts` - regenerate immediately after OpenAPI update.
- `web/src/routes/_app/settings/providers.tsx` - remove old flat provider model controls/fields that would break after contract removal.
- `web/src/hooks/routes/use-settings-providers-page.ts` - remove old flat provider settings view-model fields.
- `web/src/systems/settings/*` - remove old field fixtures, schemas, adapters, and tests as needed for the hard cut.

### Dependent Files
- `internal/config/provider_test.go` - builtin/validation/merge tests must move to new model shape.
- `internal/config/config_test.go` - config file parse/overlay tests using old keys.
- `internal/config/persistence_test.go` - persisted config writer assertions.
- `internal/config/bootstrap_test.go` - bootstrap output must not emit old keys.
- `internal/api/core/settings_internal_test.go` - empty settings payload behavior changes.
- `web/src/systems/session/hooks/use-session-create-dialog.ts` - must stop depending on `supported_models`; full catalog behavior lands in Task 09.
- `sdk/typescript/src/generated/contracts.ts` - regenerate if settings/session provider contract changes affect SDK types.

### Related ADRs
- [ADR-002: Provider Model Config Hard Cut](adrs/adr-002-provider-model-config-hard-cut.md) - requires deletion of old fields with no aliases.

### Web/Docs Impact
- `web/`: this task updates `web/src/generated/agh-openapi.d.ts`, `web/src/routes/_app/settings/providers.tsx`, `web/src/hooks/routes/use-settings-providers-page.ts`, `web/src/systems/settings/*`, and minimal session dialog fallback code so no old-field consumer remains after contract removal. Task 09 builds the full catalog UX.
- `packages/site`: provider/config docs still document old keys and must be updated in Task 10, but generated contract drift must not be left for Task 10.

### Extensibility / Agent Manageability / Config Lifecycle
- Extensibility: extension provider surfaces consume settings/config payloads indirectly; no new extension method in this task.
- Agent manageability: `agh config` and settings APIs must expose the new shape and reject old keys deterministically.
- Config lifecycle: removes `providers.<id>.default_model`, `providers.<id>.supported_models`, `providers.<id>.supports_reasoning_effort`; adds `providers.<id>.models.default`, `providers.<id>.models.curated`, `[model_catalog.sources.models_dev]`, and `providers.<id>.models.discovery`.

## Deliverables
- New provider model config structs, validation, merge, clone, render, and builtin defaults.
- Old flat provider model fields deleted from config/settings/backend payload code.
- OpenAPI, generated TypeScript contracts, and minimum web/settings consumers updated in the same task.
- Deterministic hard-cut errors for old TOML keys.
- Unit tests with 80%+ coverage for changed config/settings behavior **(REQUIRED)**.
- Integration-style config load/render tests for global/workspace overlays **(REQUIRED)**.

## Tests
- Unit tests:
  - [x] `providers.codex.models.default` parses and survives clone/merge/render.
  - [x] `models.curated` rejects blank `id` values and duplicate model IDs.
  - [x] `default_reasoning_effort` outside `reasoning_efforts` returns an exact validation error.
  - [x] old `default_model` returns a deterministic hard-cut error path.
  - [x] old `supported_models` returns a deterministic hard-cut error path.
  - [x] old `supports_reasoning_effort` returns a deterministic hard-cut error path.
  - [x] manual `models.default` outside `models.curated` is accepted.
  - [x] `model_catalog.sources.models_dev` validates enabled/endpoint/TTL/timeout defaults and rejects invalid durations or URLs.
  - [x] provider `models.discovery` rejects unsafe or ambiguous command/endpoint configuration.
  - [x] builtin `claude`, `codex`, `pi`, and gateway providers expose defaults through the new shape.
- Integration tests:
  - [x] global + workspace overlay merge preserves explicit curated model metadata.
  - [x] config persistence writes only the new nested shape.
  - [x] settings update paths can set, clear, and list nested model config without old fields.
  - [x] generated OpenAPI and `web/src/generated/agh-openapi.d.ts` no longer expose old provider model fields.
  - [x] web settings tests compile and pass against the new generated provider settings payload.
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- No production/test reference to `ProviderConfig.DefaultModel`, `SupportedModels`, or `SupportsReasoningEffort` remains outside historical docs or deliberate hard-cut error tests.
- Old TOML keys are rejected instead of silently ignored or translated.
- `make codegen-check`, `make bun-typecheck`, and focused settings tests pass after the hard cut.

## Verification Evidence
- `rtk make codegen` passed.
- `rtk make codegen-check` passed.
- `rtk make bun-typecheck` passed.
- Focused web settings/session Vitest command passed: 5 files, 66 tests.
- `rtk go test ./internal/config` passed after the final self-review correction: 622 tests.
- `rtk make verify` passed after all code changes: exit code 0.
