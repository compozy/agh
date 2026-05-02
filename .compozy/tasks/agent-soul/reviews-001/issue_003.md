---
provider: coderabbit
pr: "88"
round: 1
round_created_at: 2026-05-02T18:22:40.559088Z
status: valid
file: internal/api/contract/automation.go
line: 83
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM5_IrdG,comment:PRRC_kwDOR5y4QM69XbzF
---

# Issue 003: _⚠️ Potential issue_ | _🟠 Major_ | _🏗️ Heavy lift_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_ | _🏗️ Heavy lift_

**Avoid baking raw webhook secret bindings into the public trigger contract.**

`webhook_secret_ref` is now part of both the trigger payload and the transport request DTOs, which leaks secret-binding identifiers to API/CLI/web consumers and standardizes them on the wire. This should be modeled as opaque/presence metadata instead of exposing the raw binding.
 

As per coding guidelines, "Raw `claim_token` (`agh_claim_*`), MCP auth tokens, OAuth codes, PKCE verifiers, and secret bindings must NEVER appear in logs, status APIs, settings views, error payloads, channel messages, SSE, web UI, or memory; use hash forms (`claim_token_hash`) over the wire".


Also applies to: 151-183

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/api/contract/automation.go` around lines 64 - 83, The TriggerPayload
currently exposes the raw webhook binding via the webhook_secret_ref field (and
similarly in the transport request DTOs), which leaks secret identifiers; remove
webhook_secret_ref from TriggerPayload and the request DTOs and replace it with
an opaque presence indicator and/or a hashed form (e.g., WebhookSecretPresent
bool and WebhookSecretHash string) so callers only learn that a secret is
configured (and an irreversible hash) instead of the raw binding; update all
places that populate or marshal TriggerPayload and the transport DTOs to compute
and set the hash server-side (never accept or propagate the raw binding), change
JSON tags accordingly (e.g., "webhook_secret_present" / "webhook_secret_hash"),
and ensure logging, error messages, and any serialization paths do not include
the raw value (search for usages of TriggerPayload.webhook_secret_ref and the
transport request DTO names to update producers/consumers).
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Root cause: `contract.TriggerPayload` exposes `WebhookSecretRef` as `webhook_secret_ref`, and the shared HTTP/UDS/CLI transport create/update DTOs accept `webhook_secret_ref`. `internal/api/core/conversions.go` currently copies the raw persisted secret binding into response payloads, so callers can observe a vault/env binding identifier through public automation trigger reads.
- Fix approach: remove raw `webhook_secret_ref` from public trigger response/request DTOs, replace response output with `webhook_secret_present` plus `webhook_secret_hash`, and compute that hash from the persisted ref in the server-side conversion layer. Creation/update over transport will keep the write-only `webhook_secret_value`; the daemon will continue deriving the owned vault ref server-side through the existing automation manager path. This requires touching generated OpenAPI/web types, converters, and focused tests outside the single listed code file because the contract change would otherwise leave producers/consumers uncompilable.
