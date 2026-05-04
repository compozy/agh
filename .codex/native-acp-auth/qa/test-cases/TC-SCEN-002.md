# TC-SCEN-002: Bound-Secret Providers Still Enforce Required Credentials

**Priority:** P0

## Objective

Validate that AGH-managed API-key providers still fail fast when required `credential_slots` cannot
resolve a secret.

## Preconditions

- Provider config includes `pi_acp` providers such as `openrouter` or a custom
  `auth_mode = "bound_secret"` provider.
- Required secret refs are absent.

## Test Steps

1. Resolve the bound-secret provider.
   **Expected:** The provider resolves with `auth_mode = bound_secret` and at least one required
   credential slot.
2. Prepare provider startup with the required `env:` or `vault:` ref missing.
   **Expected:** Startup preparation fails before subprocess launch with a missing required secret
   error.
3. Inspect settings auth status for the provider.
   **Expected:** The status is redacted and machine-readable as missing required credential.

## Behavioral Evidence

Operator journey: a gateway provider cannot launch until its API key is intentionally supplied by
the service manager or AGH Vault.

Cross-surface assertions: CLI auth status, settings API, and web Settings report missing required
state without leaking secret values.

## Disruption Probes

- Change the provider to `auth_mode = "none"` while leaving `credential_slots` configured.
  **Expected:** Config validation rejects the contradictory shape.
