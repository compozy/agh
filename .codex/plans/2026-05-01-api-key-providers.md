# API-Key Provider Support via Pi Harness and Bound Vault Injection

## Summary

Implement secondary providers such as OpenRouter, z.ai, Moonshot/Kimi, Vercel AI Gateway, MiniMax, DeepSeek, xAI, Ollama, and custom OpenAI-compatible endpoints as first-class AGH providers while keeping the execution boundary ACP-compatible.

The v1 implementation uses a canonical provider catalog, routes API-key providers through `pi-acp`, materializes an isolated Pi runtime configuration per session, and injects credentials through AGH-owned bound secret resolution. This follows the strongest patterns found in Hermes, GoClaw, OpenClaw/OpenFang, Compozy Code, Pi, and AGH ADR-008: visible provider identity, reusable runtime harness, encrypted secret storage, and no generic `vault/get` access.

## Key Changes

- Persist analysis artifacts under `.compozy/tasks/providers/analysis/`:
  - `analysis_agh_current.md`
  - `analysis_hermes.md`
  - `analysis_openclaw_openfang.md`
  - `analysis_goclaw_vault.md`
  - `analysis_compozy_code_pi.md`
  - `analysis_web_site_docs.md`
- Add a provider catalog layer:
  - Keep existing ACP providers as command-backed providers.
  - Add catalog descriptors for secondary providers with stable public IDs, display names, aliases, default models, runtime harness, runtime provider ID, credential slots, transport/API mode, optional base URL, and readiness checks.
  - Separate public provider identity from execution harness.
- Make model and provider routing real:
  - Extend resolved provider/session state so `default_model`, selected model, runtime provider, and harness are carried into session start.
  - For `pi-acp` providers, generate an AGH-owned Pi runtime directory and set `PI_CODING_AGENT_DIR` for the child process.
  - Generate Pi settings and optional custom Pi `models.json`.
  - Preserve selected AGH provider ID in session metadata/events.
- Add AGH Secret Vault for provider credentials:
  - Implement encrypted secret storage aligned with AGH ADR-008: bound secret injection, not arbitrary secret browsing.
  - Use AES-256-GCM with mandatory encryption and no plaintext fallback.
  - Store provider config separately from secret material.
  - Support `env:NAME` and `vault:providers/<provider>/<slot>` references.
  - Resolve only the selected provider's bound credential slots at session spawn and inject only declared target env vars.
  - Add dynamic redaction seeded with resolved secret values.
- Extend backend contracts and management surfaces:
  - Extend provider settings payloads with catalog/runtime/readiness/credential metadata.
  - Add provider credential write APIs that accept secret values but return only masked/status DTOs.
  - Add provider verification modes.
  - Mirror management across HTTP, UDS, and CLI.
- Update `web/`:
  - Replace command/API-key-env-only settings with catalog-driven provider settings.
  - Show identity, harness, credential state, readiness, default model, base URL, verification result, and warnings.
  - Add write-only secret entry and masked credential state.
  - Update session creation provider picker readiness behavior.
- Update `packages/site` and copywriting:
  - Document first-class AGH provider identity backed by Pi runtime for API-key providers.
  - Document provider catalog fields, credential bindings, env refs, vault refs, default models, custom base URLs, verification, CLI, and API behavior.

## Public APIs and Interfaces

- Provider config gains catalog/runtime fields:
  - `display_name`
  - `harness`: `acp` or `pi_acp`
  - `runtime_provider`
  - `default_model`
  - `aliases`
  - `base_url`
  - `transport`
  - `credential_slots`
  - `mcp_servers`
  - readiness metadata
- Provider credential slots use explicit bindings:
  - `slot`
  - `target_env`
  - `required`
  - `secret_ref`: `env:OPENROUTER_API_KEY` or `vault:providers/openrouter/api_key`
- Session create/resume stores the selected AGH provider ID plus resolved runtime metadata.
- Settings and workspace provider responses return masked/status-only credential information.

## Test Plan

- Backend unit tests:
  - Provider catalog normalization, aliases, defaults, validation, and readiness.
  - Pi runtime materialization.
  - Vault encryption/decryption, key loading, missing/locked vault behavior, env refs, vault refs, redaction, and no plaintext response DTOs.
  - Masked update behavior.
  - Provider verification modes.
- Backend integration/E2E:
  - Fake `pi-acp`/fake Pi binaries capture env/config files.
  - Session metadata preserves public provider identity.
  - Logs/SSE/transcripts redact injected secrets.
  - Resume behavior with present, missing, and rotated credentials.
  - Schema migration tests.
- Web tests:
  - Provider settings render catalog-backed providers, credential state, readiness, verification results, and masked secret update behavior.
  - Session creation picker explains unavailable providers.
- Site/docs checks:
  - Regenerate CLI docs if CLI changes.
  - Run package/site checks and full monorepo verification.
- Required final gate:
  - `make codegen` if contracts change.
  - `make codegen-check`
  - `make bun-lint`
  - `make bun-typecheck`
  - `make bun-test`
  - `make web-build`
  - `make fmt`
  - `make lint`
  - `make test`
  - `make build`
  - `make boundaries`
  - `make verify`

## Assumptions and Defaults

- V1 keeps AGH's process boundary ACP-compatible by using `pi-acp` for API-key providers instead of implementing native cloud transport clients in Go.
- V1 does not expose arbitrary vault reads to extensions, agents, or Host APIs.
- Encryption is mandatory because AGH is greenfield alpha and has no legacy plaintext rows to support.
- `env:NAME` remains supported for operator-managed secrets, but AGH-managed provider setup defaults to encrypted vault refs.
- Credential pooling, OAuth/device-code provider auth, external KMS, and provider-side key rotation are deferred.
- `pi-acp` MCP pass-through remains documented as unsupported or limited until verified with real runtime evidence.
