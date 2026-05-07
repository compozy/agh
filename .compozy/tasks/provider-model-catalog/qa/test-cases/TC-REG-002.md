# TC-REG-002: Generated Docs and CLI Reference Stay in Sync

**Priority:** P1
**Type:** Regression
**Surface:** `packages/site`, `make cli-docs`, `make codegen-check`.
**Requirement:** TechSpec Docs Impact, Task 10.
**Status:** Not Run

## Objective

Verify generated CLI docs, generated OpenAPI/TS types, and narrative MDX align with current production behavior.

## Preconditions

- [ ] Branch up-to-date.

## Test Steps

1. **Run `make cli-docs`.**
   - **Expected:** No diff against committed `packages/site/content/runtime/cli/provider/models/{list,refresh,status}.mdx`.
2. **Run `make codegen-check`.**
   - **Expected:** No diff in `openapi/agh.json` or `web/src/generated/agh-openapi.d.ts`.
3. **Run `cd packages/site && bun run test -- provider-model-catalog-docs`.**
   - **Expected:** Suite passes; no flat-field claims outside warning copy.
4. **Open `packages/site/content/runtime/core/agents/model-catalog.mdx`.**
   - **Expected:** Documents native HTTP/UDS catalog endpoints, `/api/openai/v1/models`, refresh lifetime/coalescing, extension `model.source`. Merge priority table reflects current source priorities (config 120 / live 110 / extension 100 / models_dev 50 / builtin 10).
5. **Open `packages/site/content/runtime/core/configuration/config-toml.mdx`.**
   - **Expected:** `[model_catalog.sources.models_dev]`, `models.discovery`, and nested `[providers.<id>.models]` documented; defaults and validation rules match TechSpec.

## Audit Coverage

- C6 (Task 10).

## Pass Criteria

- All gates green; no diff.

## Failure Criteria

- Codegen diff.
- Docs vitest fails.
- Narrative copy contradicts daemon behavior.
