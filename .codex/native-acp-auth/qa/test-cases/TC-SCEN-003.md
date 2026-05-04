# TC-SCEN-003: Provider Env And Home Isolation Preserve Auth Boundaries

**Priority:** P1

## Objective

Validate that provider launch applies `env_policy` and `home_policy` consistently for runtime
sessions and CLI auth commands.

## Preconditions

- A native provider is configured with `env_policy = "isolated"` or `home_policy = "isolated"`.
- The parent daemon environment contains secret-shaped variables.

## Test Steps

1. Build the provider launch environment with `env_policy = "isolated"`.
   **Expected:** Secret-shaped daemon variables are absent unless explicitly injected by a
   bound-secret slot.
2. Build the provider launch environment with `home_policy = "isolated"`.
   **Expected:** `$AGH_HOME/providers/<provider>` is created with private permissions and
   `PROVIDER_HOME`, `HOME`, XDG directories, and known provider-specific home variables point there.
3. Run provider auth status/login command preparation for the same provider policy.
   **Expected:** CLI auth commands receive the same env/home policy as session launch.

## Behavioral Evidence

Operator journey: an operator can choose between reusing existing native CLI login state and
starting with a private AGH-owned provider home.

Cross-surface assertions: session launch and `agh provider auth` use the same provider environment
policy, so diagnostics do not observe a different auth store than runtime sessions.

## Disruption Probes

- Set both `env_policy = "isolated"` and a required bound secret.
  **Expected:** The isolated env contains the explicit bound target env var and excludes unrelated
  daemon secrets.
