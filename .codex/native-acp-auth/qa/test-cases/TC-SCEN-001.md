# TC-SCEN-001: Native ACP Providers Launch Without AGH-Bound API Keys

**Priority:** P0

## Objective

Validate that direct ACP providers with native CLI authentication are not blocked by missing
provider API-key environment variables.

## Preconditions

- Provider config uses built-in direct ACP providers.
- The daemon/test environment does not provide provider API-key variables for native providers.

## Test Steps

1. Resolve built-in providers such as `claude`, `codex`, `gemini`, `opencode`, `hermes`, and
   `openclaw`.
   **Expected:** Each resolves with `auth_mode = native_cli`, `env_policy = filtered`, and
   `home_policy = operator`.
2. Prepare provider startup for native providers without provider API-key env vars.
   **Expected:** Startup preparation does not fail because of missing API-key env vars.
3. Inspect provider metadata returned through settings/session provider APIs.
   **Expected:** Native providers report native auth state and do not expose credential slots.

## Behavioral Evidence

Operator journey: an operator who already ran the provider's native login can select that provider
in AGH without also binding an API key in AGH.

Cross-surface assertions: backend config, session runtime, settings API, generated OpenAPI types,
and web Settings all represent the provider as native-auth.

## Disruption Probes

- Configure a native provider with `credential_slots` but no `auth_mode = "bound_secret"`.
  **Expected:** Config validation fails with a targeted native auth error.
