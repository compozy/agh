# ADR-001: Daemon-Owned Provider Model Catalog

## Status

Accepted

## Context

AGH currently exposes provider model hints through provider config fields (`default_model`, `supported_models`, `supports_reasoning_effort`) and captures ACP `models.availableModels` only after a session exists. That is insufficient for pre-session model selection because the web dialog, CLI, HTTP, UDS, and extensions all need the same provider/model/reasoning view before `session/new`.

Research inputs point to two distinct authorities:

- Zed uses ACP session config options for active session controls.
- Harnss combines provider-specific model paths, Codex `model/list`, Claude supported models cache, and ACP config option cache.
- Paperclip keeps adapter-specific model discovery and validation in adapter runtimes, with explicit model/manual behavior.
- `compozy-code` uses a `ModelDiscoveryService` with prioritized sources, partial success, TTL caches, stale fallback, and an OpenAI-compatible model-list projection.

## Decision

AGH will add a daemon-owned `internal/modelcatalog` service as the single authority for pre-session provider model catalog projections.

The catalog service will:

- Merge source rows keyed by `(provider_id, model_id, source_id)`.
- Persist source rows and source status in the global SQLite database.
- Query sources lazily or manually; session creation never blocks on network discovery.
- Expose identical data through HTTP, UDS, CLI, Host API, web, and HTTP-only OpenAI-compatible `/api/openai/v1/models`.
- Keep active ACP `configOptions` separate from pre-session catalog state.

## Consequences

- Provider model selection stops depending on `SessionProviderOptionPayload.supported_models`.
- Operators and agents can refresh and inspect model source health without using the web UI.
- `models.dev` becomes enrichment and stale fallback, not proof of account-level availability.
- ACP sessions retain their own session-scoped config authority; observed ACP state does not rewrite the global catalog.

## References

- `.resources/zed/crates/agent_ui/src/config_options.rs`
- `.resources/zed/crates/acp_thread/src/connection.rs`
- `.resources/harnss/electron/src/ipc/claude-sessions.ts`
- `.resources/harnss/shared/lib/codex-helpers.ts`
- `.resources/paperclip/adapter-plugin.md`
- `.resources/paperclip/packages/adapters/opencode-local/src/server/models.ts`
- `/Users/pedronauck/dev/compozy/compozy-code/providers/sdk/src/models/model-discovery-service.ts`
- `/Users/pedronauck/dev/compozy/compozy-code/providers/sdk/src/models/catalog-sources/models-dev-source.ts`
