---
status: resolved
file: internal/automation/trigger.go
line: 420
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM56TZaG,comment:PRRC_kwDOR5y4QM623-TP
---

# Issue 007: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Don't permanently reserve the delivery ID before dispatch succeeds.**

`claimWebhookDelivery()` runs before `dispatchMatches()`. If dispatch returns an error, this request will fail, but the delivery ID stays claimed for the whole freshness window, so the sender's retry gets `ErrWebhookReplayDetected` instead of another real attempt. That turns transient dispatcher failures into dropped webhooks.

A safer pattern here is an in-flight claim that is only finalized after a successful dispatch, or a rollback when dispatch fails before any run is persisted.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/automation/trigger.go` around lines 415 - 420, The current flow
calls e.claimWebhookDelivery(registration.Trigger.ID, request.DeliveryID) before
dispatchMatches, causing a permanent claim even if dispatch fails; change to
either (A) make the claim "in-flight" and finalize it only after dispatchMatches
returns successfully (e.g., call a finalizeClaim/confirmClaim method after
dispatch), or (B) keep the existing claim but immediately release/rollback it if
dispatchMatches returns an error (e.g., call an unclaim/release method in the
error path). Update the logic around webhookEnvelope(...) and
e.dispatchMatches(...) so the delivery ID is only permanently marked claimed
after a successful dispatch or is explicitly released on dispatch failure; use
the unique symbols e.claimWebhookDelivery, e.dispatchMatches, webhookEnvelope,
TriggerRegistration and ensure TriggerResult error paths unclaim when needed.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - `HandleWebhook()` claims the delivery id before dispatch, and the claim is currently permanent for the freshness window even when dispatch fails before any run is recorded.
  - That turns transient dispatcher failures such as temporary concurrency rejection into replay rejections on retry, which can drop legitimate webhook deliveries.
  - Fix approach: keep the pre-dispatch claim for replay safety, but explicitly release it again when dispatch returns an error without recording any run, then cover the retry path with a regression test.
  - Resolution: released webhook delivery claims after no-run dispatch failures, added a retry regression test for transient concurrency failure, and verified with focused `go test` runs plus `make verify`.
