# AGH Provider Model Catalog + ACP Session Config Options

## Summary

Implement a daemon-owned model catalog that powers provider/model/reasoning selection before session creation, while treating ACP `configOptions` as the source of truth for active session controls.

The design is a hard cut: remove the old provider fields `default_model`, `supported_models`, and `supports_reasoning_effort`; replace them with a nested `models` block. `models.curated` is not an allowlist. It is a curated selectable list with metadata. Manual model entry remains always allowed.

The catalog must be shared across HTTP, UDS, CLI, web, and extensions, persisted in the global SQLite database, refreshed lazily/manually, and enriched from `models.dev`, live provider sources, config, built-in defaults, ACP session observations, and extension-provided sources.

## Key Changes

- Provider config schema:
  - Replace provider-level `default_model`, `supported_models`, `supports_reasoning_effort`.
  - Add `[providers.<id>.models] default = "..."`
  - Add `[[providers.<id>.models.curated]]` entries with model metadata:
    - `id`
    - `display_name`
    - `context_window`
    - `max_output_tokens`
    - `supports_tools`
    - `reasoning_efforts`
    - `default_reasoning_effort`
    - optional cost fields.
  - Keep manual model IDs valid even when absent from curated/discovered lists.

- Model catalog service:
  - Add a daemon service that merges per-source model rows into a provider/model projection.
  - Persist source rows and source status in global SQLite with refresh timestamps, expiry/staleness, and last error.
  - Merge priority:
    - config curated metadata: explicit operator metadata wins for fields it defines;
    - live provider/extension sources: authoritative for availability;
    - `models.dev`: broad enrichment and stale fallback;
    - built-in defaults: offline bootstrap fallback;
    - ACP session observations: session-scoped only, not global pre-session truth.
  - Cache behavior is lazy/manual: use cache immediately, refresh on explicit request or background expiry, never block session creation on network discovery.

- Sources:
  - Add `models.dev` source using `https://models.dev/api.json`, with 24h source TTL and stale fallback.
  - Parse both current and legacy fields: `reasoning`, `tool_call`, `limit.context`, `limit.input`, `limit.output`, `cost`, plus aliases like `supportsReasoning`, `supports_tools`, `contextWindow`.
  - Add live sources for core providers: OpenAI/Codex, Anthropic/Claude, Gemini, OpenRouter, Vercel AI Gateway, Ollama, OpenCode.
  - Add explicit side-effect-free source adapters/config support for OpenClaw, Hermes, and Pi.
  - Do not include Droid in v1.
  - Do not create fake ACP sessions for discovery.

- ACP session config:
  - Upgrade `github.com/coder/acp-go-sdk` from `v0.6.3` to the latest available version, currently `v0.12.2`.
  - Capture `configOptions` from `session/new`, `session/load`, and `config_option_update`.
  - Prefer `session/set_config_option` for model/reasoning changes when ACP exposes config options.
  - Keep legacy `models.availableModels` / `setSessionModel` only as fallback where config options are absent.
  - Active session UI must use ACP config options over catalog metadata.

- Public surfaces:
  - Add native HTTP/UDS model catalog endpoints:
    - list all models;
    - list models by provider;
    - refresh all/provider/source;
    - read provider/source status.
  - Add OpenAI-compatible `GET /v1/models` projection with AGH metadata for interoperability.
  - Add CLI under existing provider namespace:
    - `agh provider models list [provider]`
    - `agh provider models refresh [provider]`
    - `agh provider models status [provider]`
  - Add extension support:
    - manifest capability `model.source`;
    - AGH->extension service method `models/list`;
    - extension Host API methods to list/refresh/status catalog data for extension authors/operators.
  - Regenerate OpenAPI, web generated types, extension SDK types, and CLI docs.

- Web:
  - New session dialog loads models from the catalog for the selected provider.
  - It shows curated, discovered, stale, manual, and source/error states without blocking session creation.
  - Settings > Providers edits the new `models` block and exposes discovery status + refresh.
  - After session creation, session controls switch to ACP `configOptions` if present.

## Test Plan

- Config tests:
  - Old fields are removed with no compatibility aliases.
  - New `models.default` and `models.curated` parse, merge, clone, validate, and render correctly.
  - Manual model IDs remain accepted when absent from curated/discovered lists.

- Catalog service tests:
  - SQLite migration creates model/source/status tables.
  - Fresh DB, reopen-after-restart, stale cache, refresh failure, and partial-source success work correctly.
  - Merge priority preserves explicit config metadata, live availability, `models.dev` enrichment, and built-in fallback.

- Source tests:
  - `models.dev` parser handles current `reasoning/tool_call/limit/cost` and legacy aliases.
  - Live provider sources use httptest/fake subprocesses in normal verify.
  - OpenClaw/Hermes/Pi adapters fail closed with clear source status when their explicit discovery command/endpoint is unavailable.
  - No discovery path creates ACP sessions.

- ACP tests:
  - SDK upgrade compiles and preserves existing session behavior.
  - `configOptions` are captured on new/load and updated on `config_option_update`.
  - `session/set_config_option` is preferred for model/reasoning when available.
  - Legacy model state fallback remains covered.

- Surface tests:
  - HTTP, UDS, CLI, and Host API return matching catalog/status data.
  - `/v1/models` projection is stable and includes AGH metadata.
  - Web hooks/components cover loading, stale/error state, refresh, manual entry, and ACP post-session override.

- Verification:
  - `make codegen` after contract changes.
  - Focused Go/web tests during implementation.
  - Final `make verify`.
  - Real provider discovery tests are opt-in with explicit env/tags and are not required by `make verify`.

## Assumptions And Defaults

- This is greenfield: hard cut old fields, no fallback aliases, no compatibility migrations for obsolete config keys.
- `models.curated` is a curated list with metadata, not a permission boundary.
- Manual model entry is always allowed.
- Discovery uses the provider's effective auth/home/env policy.
- Live discovery must be side-effect-free and timeout-bound.
- `models.dev` enriches and broadens the catalog but does not prove account-level availability.
- Active ACP session `configOptions` override catalog assumptions for that session.
- Droid is explicitly out of scope for v1.
