# Claude Opus Remediation Review: API-Key Providers

Command:

```bash
compozy exec --ide claude --model opus --reasoning-effort xhigh --timeout 20m --prompt-file .compozy/tasks/providers/reviews/claude_opus_remediation_review_prompt.md
```

## Findings

### 1. MAJOR — Daemon "vault disabled" warning leaves a typed-nil resolver wired through session + settings

**Files**: `internal/daemon/boot.go:486–490, 644, 1559`, `internal/daemon/daemon.go:550`, `internal/session/manager.go:352–354`, `internal/settings/collections.go:441–443`.

`buildProviderVault` returns `(*vault.Service)(nil), nil` when the registry doesn't implement `vault.Store` (`boot.go:653–658`). `state.providerVault` is the concrete pointer type `*vault.Service` (`boot.go:67`). It is then assigned to `SessionManagerDeps.ProviderSecrets` (interface `session.ProviderSecretResolver`, `daemon.go:293`) and `settingspkg.Dependencies.ProviderSecrets` (interface `settings.ProviderSecretStore`). Storing a typed-nil pointer in an interface produces a non-nil interface.

Concrete impact:

- `applyRuntimeDefaults` in `manager.go:352` checks `if m.providerSecrets == nil { m.providerSecrets = envProviderSecretResolver{...} }`. The typed-nil check fails, so the env-only fallback never engages.
- The first session start that calls `m.providerSecrets.ResolveRef(ctx, ref)` dispatches to `(*vault.Service)(nil).ResolveRef(...)`, which dereferences `s.lookupEnv` / `s.store` and panics.
- Same hazard in `settings/collections.go:441`; a `PUT /settings/providers/...` with `secrets[]` panics in `(*vault.Service)(nil).PutSecret(...)` via `s.keys.Key()`.
- `internal/daemon/provider_vault_test.go` only asserts the warning is logged; it never exercises a session start or settings PUT through the disabled path, so the panic is invisible to the suite.

Production today is shielded only because `*globaldb.GlobalDB` implements `vault.Store`. The remediation's contract — "warn and continue gracefully" — is not honored.

**Fix**: coerce the typed-nil to a true nil interface before propagation, or change `buildProviderVault` to return the interface type and explicitly return a nil-typed interface. Add a daemon-level test that boots with a non-vault registry and starts a session with credential slots, asserting graceful env-only behavior.

### 2. SIGNIFICANT — `Required: true` is silently dropped for env-bound credentials on direct ACP providers

**Files**: `internal/session/provider_runtime.go:138–154`, `internal/session/provider_runtime_test.go:191–198`, `packages/site/content/runtime/core/agents/providers.mdx:71`, `packages/site/content/runtime/core/agents/spawning.mdx:96–100`.

```go
if vault.IsEnvRef(secretRef) {
    return errors.Is(err, vault.ErrMissingSecret)
}
```

This branch ignores `slot.Required`. The new test pins `Required: true` to skip for direct ACP providers. Operators who explicitly set `required: true` on a `claude`/`codex`/`gemini` slot will silently launch with the binding absent.

**Fix**: tighten to `return !slot.Required && errors.Is(err, vault.ErrMissingSecret)` for env refs as well, then update the test matrix. If lenient behavior is intentional, document the divergence.

### 3. SIGNIFICANT — `materializePiRuntime` writes the env-var name into Pi `models.json.apiKey`

**Files**: `internal/session/provider_runtime.go:215, 226–240`, `internal/session/provider_runtime_test.go:94–96`.

`piCredentialEnv` returns the env variable name and that string is serialized as `models.json.apiKey`. Pi v0.0.26 is expected to env-resolve that field, but the test only pins env-name presence; there is no integration evidence that Pi treats `apiKey` as an env reference rather than a literal token.

**Fix**: add a fake `pi-acp` E2E that parses `models.json` and asserts the contract, or pivot `apiKey` to the resolved secret value and rely on dynamic redaction plus 0o600 file permissions.

### 4. MINOR — Daemon vault warning omits the registry implementation type

**File**: `internal/daemon/boot.go:653–658`.

The warning does not include the registry implementation, so an operator has little signal about which registry produced it.

**Fix**: add `"registry_type", fmt.Sprintf("%T", state.registry)`.

### 5. MINOR — Provider-runtime tests skip file mode, transport/base_url, and slot-priority assertions

**File**: `internal/session/provider_runtime_test.go`.

Missing assertions include 0o600 file mode, multi-slot `api_key` priority, and `BaseURL`/`Transport` propagation into `models.json`.

**Fix**: extend the existing tests.

### 6. MINOR — Vault upsert idempotency and metadata-update semantics are not asserted

**File**: `internal/store/globaldb/global_db_vault_test.go`.

No test calls `PutVaultSecret` twice with different `EncryptedValue`/`UpdatedAt` and verifies the latest ciphertext and timestamp win.

**Fix**: add an upsert subtest.

### 7. NIT — `validProviderSecretRef` is asymmetric on case

**File**: `internal/config/provider.go:298`.

The provider segment is lowercase only, but the slot segment allows uppercase. Docs do not surface the exact pattern.

**Fix**: tighten the slot segment to lowercase, or document the exact pattern.

## Non-Blocking Improvements

- Add a comment or test for the dynamic redaction minimum length threshold.
- Consider moving provider redaction cleanup into session-start failure handling for panic symmetry.
- Change the daemon provider-vault boundary to an interface to make typed-nil propagation impossible.
- Consider redacting exact `vault:` refs in settings responses for support/screensharing contexts.

## Verification Gaps

- Settings HTTP-layer test that `PUT /api/settings/providers/<name>` with `secrets[]` returns no `value` and `GET /api/settings/providers` reports `present=true` without echoing the value.
- Fake `pi-acp` binary E2E that captures `PI_CODING_AGENT_DIR`, asserts `OPENROUTER_API_KEY` is set in the child process, and asserts no plaintext secret appears in stderr or persisted session-failure events.
- Daemon-level integration test that boots with a non-vault registry and attempts a provider session start or settings mutation.
- Snapshot guard between the docs provider table and `BuiltinProviders()`.

## Readiness Verdict

**Not ready** — address Findings 1–3 before treating the remediation as complete.
