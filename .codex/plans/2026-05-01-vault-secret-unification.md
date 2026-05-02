# Vault as the Canonical Durable Secret Authority

## Summary

Unify AGH's durable secret handling around `internal/vault` so providers/ACP, bridges, automation webhooks, MCP auth, MCP stdio env secrets, hooks/extensions/sandbox secret env bindings all use the same `env:` / `vault:` reference model.

The implementation will hard-cut ambiguous public names and parallel stores: `vault_ref` becomes `secret_ref`, env-only secret fields become `*_secret_ref`, raw durable secret payloads become write-only vault secret writes, and durable token/secret persistence moves behind the vault crypto/key authority. Ephemeral bearer material such as claim tokens and approval tokens stays outside vault, but remains hash/redaction-only.

External security baseline: OWASP recommends centralized secret storage/provisioning/rotation, avoiding secrets in config/env where possible, and never logging access tokens, passwords, encryption keys, or primary secrets.

## Key Changes

- Generalize the current provider-only vault wiring into a daemon-wide secret service.
  - Keep `env:NAME` as operator-managed lookup and `vault:<namespace>/...` as AGH-managed encrypted storage.
  - Add shared validation helpers for allowed vault namespaces: `providers`, `bridges`, `automation`, `mcp`, `hooks`, `extensions`, and `sandbox`.
  - Ensure plaintext resolution is only available to internal runtime materialization boundaries; public APIs return metadata/status only.
  - Register every resolved plaintext value with diagnostics dynamic redaction before it can appear in process env, errors, logs, status, SSE, or tool output.
- Remove provider credential ambiguity.
  - Keep `credential_slots[].secret_ref` as the only provider credential model.
  - Remove `api_key_env` shortcut/fallback generation from config, settings, docs, generated contract, and web state.
  - Tighten provider secret writes so settings can store only refs valid for that provider namespace, not arbitrary `vault:` refs.
  - Update the providers settings UI to support all credential slots, not only the first slot.
- Hard-cut bridge secret bindings.
  - Rename bridge contract/config/type field `vault_ref` / `VaultRef` to `secret_ref` / `SecretRef`.
  - Delete the stock daemon's env-only bridge resolver and resolve bridge bindings through the shared vault service.
  - Add write-only bridge secret writes for `vault:bridges/<bridge-instance>/<binding>` refs; never return secret values.
  - Update HTTP/UDS contracts, generated OpenAPI/TypeScript, web bridge drafts/forms, tests, and docs.
- Move automation webhook secrets to vault.
  - Replace config `webhook_secret_env` with `webhook_secret_ref`.
  - Replace raw API/tool/extension `webhook_secret` fields with `webhook_secret_ref` plus write-only secret write payloads.
  - Store webhook material under deterministic vault refs such as `vault:automation/triggers/<trigger-id>/webhook-secret`.
  - Delete the plaintext `automation_trigger_webhook_secrets` table and its store methods; webhook verification resolves the ref only at signature-check time.
- Move MCP durable secrets to vault authority.
  - Replace MCP OAuth `client_secret_env` with `client_secret_ref`.
  - Replace MCP stdio secret-like raw env values with explicit secret env bindings, e.g. `secret_env = { GITHUB_TOKEN = "vault:mcp/<server>/env/GITHUB_TOKEN" }`.
  - Keep literal `env` only for non-secret operational values; validators reject secret-looking keys in literal env maps.
  - Remove the separate `.mcp-auth.key` crypto path and store OAuth access/refresh token material through the vault service, using refs such as `vault:mcp/<server>/oauth/access-token` and `vault:mcp/<server>/oauth/refresh-token`.
  - Preserve public MCP auth status as redacted metadata only.
- Add secret env bindings for other runtime env surfaces.
  - Introduce the same literal-env vs secret-env split for hooks, extensions, and sandbox profiles.
  - Resolve secret env refs only when launching or invoking the relevant runtime boundary.
  - Keep `{{env:NAME}}` expansion only for non-secret operator-managed values; use explicit secret refs for secret material.
  - Reject docs/examples/config that place API keys, tokens, passwords, or secrets directly into literal env maps.
- Keep ephemeral token behavior separate.
  - Do not move task claim tokens, tool approval tokens, OAuth transient codes, or PKCE verifiers into vault.
  - Continue storing only hash/redacted forms where persistence or reporting is required.
- Preserve AGH agent-manageability.
  - Expose CLI/HTTP/UDS parity for secret metadata, write-only create/update, delete, and status across providers, bridges, automation, and MCP.
  - Native tools must never accept or return raw durable secret material except through explicit write-only secret payloads.
  - Update config tool guardrails so new secret paths (`secret_ref`, `client_secret_ref`, `webhook_secret_ref`, `secret_env`) are treated as trust-root/secret paths.

## Public Interfaces / Types

- Rename/remove public contract fields with no aliases:
  - `bridges.*.vault_ref` -> `secret_ref`
  - `mcp.auth.client_secret_env` -> `client_secret_ref`
  - `automation.triggers.*.webhook_secret_env` -> `webhook_secret_ref`
  - automation API/tool/extension raw `webhook_secret` -> write-only secret write payload + `webhook_secret_ref`
  - provider `api_key_env` -> removed; use `credential_slots[].secret_ref`
  - secret-like `env` values -> `secret_env` bindings on MCP/hooks/extensions/sandbox surfaces
- Regenerate and co-ship:
  - `openapi/agh.json`
  - generated web OpenAPI types
  - web API hooks/components affected by settings, bridges, MCP, and automation
  - site docs and CLI/config references
- Schema hard cut:
  - Keep `vault_secrets` as the durable encrypted secret table.
  - Remove plaintext automation webhook secret persistence.
  - Remove MCP auth's independent encrypted-token/key storage and persist token material through vault-backed records.
  - Because AGH is greenfield alpha, do not add compatibility readers, aliases, or fallback paths for deleted fields.

## Test Plan

- Vault unit tests:
  - Validate all namespace patterns and reject malformed refs.
  - Confirm `PutSecret` accepts only valid `vault:` refs and `ResolveRef` handles `env:` / `vault:` consistently.
  - Confirm metadata/list/delete never expose plaintext.
- Provider tests:
  - Prove providers work only through `credential_slots[].secret_ref`.
  - Prove `api_key_env` is rejected/removed from config, settings, docs fixtures, and generated contracts.
  - Cover multi-slot provider secret writes from settings.
- Bridge tests:
  - Prove `secret_ref` accepts `env:` and `vault:` and rejects old `vault_ref`.
  - Prove bridge subprocess/env materialization receives the resolved value and redaction catches it.
  - Prove bridge API/status returns only redacted metadata.
- Automation tests:
  - Prove webhook creation/update stores no plaintext in automation tables.
  - Prove webhook verification resolves the vault ref at request time and fails closed for missing required secrets.
  - Prove native tools/extensions cannot pass or receive raw webhook secrets outside write-only secret payloads.
- MCP tests:
  - Prove `client_secret_ref` resolution for remote MCP OAuth.
  - Prove OAuth access/refresh tokens are encrypted through vault authority and `.mcp-auth.key` is no longer used.
  - Prove stdio MCP secret env bindings inject secrets at launch while literal `env` rejects secret-looking keys.
- Cross-cutting security tests:
  - Scan API responses, logs, diagnostics, SSE/status payloads, settings payloads, native tool output, and web fixtures for known sentinel secret values.
  - Confirm dynamic redaction is registered for every resolved secret.
  - Confirm config mutation tools classify all new secret paths as protected.
  - Run focused Go/web tests during implementation, then `make codegen`, `make codegen-check`, `make bun-lint`, `make bun-typecheck`, `make bun-test`, `make lint`, `make test`, `make build`, and final `make verify`.

## Assumptions

- Accepted user choices: apply the full durable-secret scope now and use hard-cut API renames with no aliases.
- Durable secrets include provider credentials, bridge bindings, automation webhook secrets, MCP client secrets, MCP OAuth tokens, and runtime secret env bindings.
- Ephemeral tokens remain outside vault by design because vault is for durable secret material, not short-lived authorization proofs.
- The first implementation step after leaving Plan Mode is to persist this accepted plan under `.codex/plans/`, then re-run a repo-wide non-mutating sweep before editing.
- Subagent exploration completed for the current vault/ACP implementation; two additional read-only subagents timed out and were closed, so implementation must begin with a fresh focused sweep over `secret`, `token`, `password`, `api_key`, `client_secret`, `webhook_secret`, `vault_ref`, and `env` surfaces before applying changes.
