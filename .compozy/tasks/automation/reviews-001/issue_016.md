---
status: resolved
file: internal/automation/trigger.go
line: 108
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM56TB0X,comment:PRRC_kwDOR5y4QM623e7k
---

# Issue 016: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Add replay protection for authenticated webhooks.**

A captured `(timestamp, signature, payload)` can be resent repeatedly within the freshness window and every copy will dispatch again. There is no delivery ID or deduplication check in `WebhookRequest`/`HandleWebhook`, so HMAC + timestamp alone still leaves this endpoint replayable.




Also applies to: 379-405

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/automation/trigger.go` around lines 99 - 108, The webhook handler is
vulnerable to replay because WebhookRequest lacks a delivery identifier and
HandleWebhook only validates HMAC+timestamp; add an explicit DeliveryID (string)
field to WebhookRequest and make HandleWebhook verify that DeliveryID is
present, check a durable/fast dedup store (e.g., Redis or in-memory cache with
TTL) for prior processing, and atomically mark the DeliveryID as seen with an
expiration equal to your freshness window to reject duplicates; ensure the
signature validation still runs and that HandleWebhook returns a clear error for
already-seen DeliveryIDs so replays are dropped.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes: HMAC plus timestamp freshness does not prevent replay of the same authenticated webhook inside the accepted window, and the current transport/runtime contract has no delivery identifier or dedup tracking. I will add an explicit delivery identifier to the webhook request flow, reject missing IDs, track recently seen deliveries inside the trigger runtime with freshness-window TTL, and return a clear replay error on duplicates.
