---
provider: coderabbit
pr: "88"
round: 1
round_created_at: 2026-05-02T18:22:40.559088Z
status: pending
file: internal/automation/trigger.go
line: 681
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM5_IrdX,comment:PRRC_kwDOR5y4QM69XbzW
---

# Issue 015: _⚠️ Potential issue_ | _🟠 Major_ | _⚡ Quick win_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_ | _⚡ Quick win_

**Fail webhook secret misconfiguration at registration time.**

This helper now turns an empty/invalid `WebhookSecretRef` into a runtime webhook failure, but the registration path shown in this change does not reject that configuration up front. That means a trigger can register successfully and only start failing once real deliveries arrive, which makes the misconfig much harder to detect and recover from.

Please mirror the non-empty / namespace validation during `Register` / `Update` so broken webhook triggers are rejected before they go live.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/automation/trigger.go` around lines 658 - 681, The Register/Update
paths currently allow triggers with empty or invalid WebhookSecretRef to be
saved and only fail at delivery; update TriggerEngine.Register and
TriggerEngine.Update to perform the same upfront validation as
resolveWebhookSecret: ensure Trigger.WebhookSecretRef is non-empty (trimmed),
call vault.ValidateRefNamespace(ref, "automation") and return
ErrWebhookSecretRequired (or wrap it) on failure, and verify e.webhookSecrets is
present before accepting the trigger; use the same error semantics as
resolveWebhookSecret so misconfigured webhook triggers are rejected at
registration time.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `UNREVIEWED`
- Notes:
