---
provider: coderabbit
pr: "88"
round: 1
round_created_at: 2026-05-02T18:22:40.559088Z
status: pending
file: internal/api/core/conversions.go
line: 654
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM5_IrdP,comment:PRRC_kwDOR5y4QM69XbzO
---

# Issue 008: _⚠️ Potential issue_ | _🟠 Major_ | _🏗️ Heavy lift_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_ | _🏗️ Heavy lift_

**Do not serialize `WebhookSecretRef` in trigger payloads.**

Line 651 sends the stored secret binding back through the trigger API. That leaks internal secret topology to every client that can read triggers, which the repo rules explicitly prohibit.
 
As per coding guidelines, "Raw `claim_token` (`agh_claim_*`), MCP auth tokens, OAuth codes, PKCE verifiers, and secret bindings must NEVER appear in logs, status APIs, settings views, error payloads, channel messages, SSE, web UI, or memory; use hash forms (`claim_token_hash`) over the wire".

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/api/core/conversions.go` around lines 636 - 654, The TriggerPayload
currently includes the sensitive WebhookSecretRef (in the return constructing
contract.TriggerPayload), which must not be serialized; modify the return so
WebhookSecretRef is not populated (set it to nil/empty string or omit it) when
building the payload in this conversion function (the code that constructs
contract.TriggerPayload from trigger and uses cloneFilter); ensure no other
conversion paths populate WebhookSecretRef for API responses and replace any
exposed bindings with a non-secret representation (e.g., a claim_token_hash) if
needed.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `UNREVIEWED`
- Notes:
