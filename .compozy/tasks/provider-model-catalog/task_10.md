---
status: completed
title: "Generated Contracts and Runtime Docs"
type: docs
complexity: high
dependencies:
  - task_01
  - task_07
  - task_08
  - task_09
---

# Task 10: Generated Contracts and Runtime Docs

## Overview
This task regenerates and documents every public contract changed by the model catalog feature. It keeps API, CLI, web types, SDK contracts, and runtime docs aligned with the hard-cut config and new catalog surfaces.

<critical>
- ALWAYS READ `_techspec.md` and every ADR before starting
- REFERENCE TECHSPEC for implementation details - do not duplicate here
- FOCUS ON "WHAT" - describe what needs to be accomplished, not how
- MINIMIZE CODE - show code only to illustrate current structure or problem areas
- TESTS REQUIRED - every task MUST include tests in deliverables
</critical>

<requirements>
- MUST run and commit contract/codegen outputs required by API, CLI, web, and SDK changes.
- MUST regenerate OpenAPI and TypeScript generated contract types.
- MUST regenerate CLI reference docs for `agh provider models`.
- MUST update runtime provider/config docs from old flat fields to the new nested `models` block.
- MUST document `[model_catalog.sources.models_dev]`, provider `models.discovery`, native model catalog endpoints, `/api/openai/v1/models`, extension `model.source`, and Host API model methods.
- MUST document why the CLI surface is `agh provider models ...` and that top-level `agh models ...` is out of scope for the MVP.
- MUST remove old docs claims for `supported_models` and `supports_reasoning_effort`.
- MUST keep docs truthful to implemented runtime behavior, not aspirational discovery support.
</requirements>

## Subtasks
- [x] 10.1 Run contract codegen and update OpenAPI/web/generated/SDK generated files.
- [x] 10.2 Regenerate CLI docs for provider model commands.
- [x] 10.3 Update provider and config TOML docs for nested model config.
- [x] 10.4 Add docs for native catalog endpoints and `/api/openai/v1/models` projection.
- [x] 10.5 Add extension author docs for `model.source`, `models/list`, and Host API model methods.
- [x] 10.6 Update docs tests/snapshots and remove old-field references.

## Implementation Details
Follow `_techspec.md` sections `Public Interfaces`, `Extensibility Integration Plan`, `Agent Manageability Plan`, and `Web/Docs Impact`. Activate `agh-contract-codegen-coship`, `documentation-writer`, `crafting-effective-readmes`, and site-specific instructions in `packages/site/CLAUDE.md`.

### Relevant Files
- `openapi/agh.json` - generated OpenAPI contract.
- `web/src/generated/agh-openapi.d.ts` - generated web contract types.
- `sdk/typescript/src/generated/contracts.ts` - generated TypeScript SDK contracts.
- `packages/site/content/runtime/cli-reference/` - generated CLI reference.
- `packages/site/content/runtime/core/agents/providers.mdx` - provider model config docs using old fields.
- `packages/site/content/runtime/core/configuration/config-toml.mdx` - config TOML docs using old fields.
- `packages/site/content/runtime/core/configuration/agent-md.mdx` - provider default model wording.
- `packages/site/content/runtime/core/agents/definitions.mdx` - agent model default wording.

### Dependent Files
- `internal/api/contract/**` - source of generated OpenAPI/types.
- `internal/cli/provider.go` - source of generated CLI docs.
- `packages/site/lib/__tests__/landing-truth.test.tsx` - may assert builtin provider model text.

### Related ADRs
- [ADR-001: Daemon-Owned Provider Model Catalog](adrs/adr-001-daemon-owned-provider-model-catalog.md) - public catalog docs.
- [ADR-002: Provider Model Config Hard Cut](adrs/adr-002-provider-model-config-hard-cut.md) - config docs must remove old fields.
- [ADR-003: Extension Model Source Contract](adrs/adr-003-extension-model-source-contract.md) - extension docs.

### Web/Docs Impact
- `web/`: generated `web/src/generated/agh-openapi.d.ts` changes; Task 09 must compile against it.
- `packages/site`: provider/config/API/CLI/extension docs listed above are directly updated.

### Extensibility / Agent Manageability / Config Lifecycle
- Extensibility: documents `model.source`, extension service `models/list`, and Host API model methods.
- Agent manageability: documents CLI, HTTP, UDS, `/api/openai/v1/models`, structured output, status, refresh, and deterministic errors.
- Config lifecycle: documents removed old keys, new nested model config, `[model_catalog.sources.models_dev]`, provider discovery config, and no compatibility path.

## Deliverables
- Regenerated OpenAPI, web generated types, SDK generated contracts, and CLI reference.
- Updated runtime docs for config, providers, model catalog API, `/api/openai/v1/models`, and extension source contract.
- Docs/tests with 80%+ relevant coverage for changed docs assertions **(REQUIRED)**.

## Tests
- Unit tests:
  - [ ] docs tests no longer find old provider model field claims.
  - [ ] generated contract tests compile with new payloads.
  - [ ] CLI docs include `provider models list|refresh|status`.
  - [ ] docs explain the `agh provider models ...` namespace choice.
  - [ ] provider config docs show nested `models` examples only.
  - [ ] config docs cover `model_catalog.sources.models_dev` and provider `models.discovery` keys/defaults.
- Integration tests:
  - [ ] `make codegen` produces no uncommitted drift after generation.
  - [ ] `make codegen-check` passes.
  - [ ] `make cli-docs` regenerates CLI reference without obsolete old-field docs.
  - [ ] `make bun-typecheck` validates generated web/SDK types.
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- `make codegen`, `make codegen-check`, `make cli-docs`, and `make bun-typecheck` pass.
- Public docs and generated contracts match implemented surfaces.
