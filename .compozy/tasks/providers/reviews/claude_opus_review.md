# Claude Opus Review: API-Key Providers via Pi Harness

Command:

```bash
compozy exec --ide claude --model opus --reasoning-effort xhigh --timeout 20m --prompt-file .compozy/tasks/providers/reviews/claude_opus_review_prompt.md
```

## Findings

### 1. BLOCKING — Plan-promised dynamic secret redaction is not implemented

- **Files**: feature missing across `internal/logger`, `internal/acp/client.go:1148-1152` (`attachStderr`), `internal/session/provider_runtime.go:69-106` (`injectProviderSecret`).
- **Impact**: The plan explicitly required _"Add dynamic redaction seeded with resolved secret values"_. No registry exists that scrubs the resolved API-key plaintext from logs, SSE, transcripts, or error wrapping. `attachStderr` already inlines raw subprocess stderr into start-failure errors; if `pi-acp` or upstream HTTP errors echo the token, it surfaces in observability surfaces and persisted session-failure rows.
- **Fix**: Add a process-wide redaction registry that `injectProviderSecret` seeds with the resolved value before launching the subprocess; wire it into the slog handler used by `internal/acp` and the stderr buffer in `acp.types`/`acp.client.attachStderr`. Drop or hash the value once the session terminates.

### 2. BLOCKING — No tests for the vault crypto/service/store

- **Files**: `internal/vault/`, `internal/store/globaldb/global_db_test.go`.
- **Impact**: The most security-critical Go code in this change is unverified. A regression could persist plaintext, accept unsupported refs, or leak metadata.
- **Fix**: Add tests for round-trip encryption, malformed payloads, `AGH_VAULT_KEY` decode modes, key-file permissions, `Service.PutSecret` rejecting `env:` refs, `ResolveRef` env/vault behavior, metadata behavior, globaldb prefix listing, delete, and idempotent upsert.

### 3. BLOCKING — No tests for `provider_runtime.go`

- **Files**: `internal/session/provider_runtime.go`.
- **Impact**: Pi runtime materialization and missing-secret decisions are shipping without direct test coverage.
- **Fix**: Add unit tests for generated JSON shape, file mode, `PI_CODING_AGENT_DIR`, base URL/transport propagation, API-key env selection, and a table matrix for `shouldSkipMissingProviderSecret`. Add fake `pi-acp` coverage that asserts the runtime dir and env injection.

### 4. MAJOR — `materializePiRuntime` writes env-var name even when AGH skipped injection

- **File**: `internal/session/provider_runtime.go`.
- **Impact**: Optional missing credentials can still write `apiKey: <TARGET_ENV>` into `models.json`, leaving Pi to fail later with a less precise error.
- **Fix**: Track which slots were actually injected. Omit `apiKey` when absent or fail startup with a structured missing-slot error.

### 5. MAJOR — Test for the builtin catalog ignores some new providers

- **File**: `internal/config/provider_test.go`.
- **Impact**: `xai`, `minimax`, `mistral`, and `groq` were not asserted in the catalog table, and there is no runtime-provider/default-model/alias coverage for every builtin.
- **Fix**: Extend builtin provider tests to cover every entry with command, harness, runtime provider, default model, API-key env, and aliases.

### 6. MAJOR — `daemon.buildProviderVault` returns `(nil, nil)` instead of failing fast

- **File**: `internal/daemon/boot.go`.
- **Impact**: If the registry does not implement `vault.Store`, daemon boot succeeds but vault-backed provider configs fail later with confusing session-start errors and settings status looks like "not stored" instead of "vault unavailable".
- **Fix**: Require `vault.Store` at boot for production registry composition or emit a structured warning that names the registry implementation.

### 7. MAJOR — provider credential slot overlays force incorrect required semantics

- **Files**: `internal/config/provider.go`, `internal/config/merge.go`.
- **Impact**: Replacing credential slots without preserving `required` and `kind` can make sessions fail unexpectedly.
- **Fix**: Require explicit `credential_slots` for provider credential changes, and preserve existing slot `required`/`kind` values where possible.

### 8. MAJOR — Optional vault refs may still inherit parent-shell env values

- **Files**: `internal/session/manager_start.go`, `internal/session/provider_runtime.go`.
- **Impact**: For a vault-backed slot that is optional and missing, AGH skips injection but keeps inherited `os.Environ()` values. This can undermine the expectation that a `vault:` ref means AGH-managed credential material only.
- **Fix**: Decide and document the contract. If `vault:` means AGH-managed only, scrub the target env before injection for vault-backed slots and add tests.

### 9. MAJOR — Settings provider editor only writes the first credential slot

- **Files**: `web/src/hooks/routes/use-settings-providers-page.ts`, `web/src/systems/settings/components/provider-card.tsx`.
- **Impact**: Multi-slot providers can lose all slots beyond the first during edits.
- **Fix**: Preserve/write the full slot array or restrict the UI to single-slot providers with a clear "managed elsewhere" path.

### 10. SIGNIFICANT — `validProviderSecretRef` accepts arbitrary vault suffixes

- **File**: `internal/config/provider.go`.
- **Impact**: Refs like `vault: ../something` are accepted and stored verbatim.
- **Fix**: Validate vault suffixes against a stable pattern and document allowed characters.

### 11. SIGNIFICANT — Docs need a temporary redaction caveat

- **Files**: `packages/site/content/runtime/core/agents/providers.mdx`, `spawning.mdx`, `config-toml.mdx`.
- **Impact**: Docs correctly say settings reads never expose secret values, but do not warn that subprocess stderr may surface in start-failure errors until dynamic redaction is implemented.
- **Fix**: Add an OperatorNote or implement redaction first, then no caveat is needed.

## Non-Blocking Improvements

- Centralize env lookup behavior between daemon-wired vault resolver and session manager test fallback.
- Make `piCredentialEnv` less order-dependent for future multi-slot providers.
- Consider returning settings collection meta for `GET /settings/providers/:name`, not only list responses.
- Comment or refine `providerCredentialSlotMaps` behavior for invalid partial slots.
- Either expose provider secret deletion through settings or remove unused `DeleteSecret` if it has no caller.

## Recommended Additional Tests

- `internal/vault/`: encryption/decryption, malformed payloads, key decode modes, key-file permissions, env/vault ref behavior.
- `internal/store/globaldb/global_db_vault.go`: upsert, get not found, prefix list, delete, race-sensitive put.
- `internal/session/provider_runtime.go`: JSON shape, permissions, `PI_CODING_AGENT_DIR`, injection matrix.
- `internal/session` integration: fake `pi-acp` binary that records env and runtime files.
- Settings HTTP API: `PUT /api/settings/providers/openrouter` with `secrets[]`, no echo of secret values, `GET` reports `present=true`.
- Provider catalog: every builtin, aliases, credential slot overlays.
- Web: multi-slot provider preservation and credential state rendering.
- Site/docs: snapshot guard between docs provider table and `BuiltinProviders()`.

## Reviewer Summary

The reviewer found no architectural objection to the Pi harness wiring, contract shape, or overlay direction, but flagged redaction safety and missing tests as the main issues to address before treating the feature as production-quality.
