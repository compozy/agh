---
status: resolved
file: internal/channels/delivery_broker.go
line: 340
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM56TBLj,comment:PRRC_kwDOR5y4QM623eI8
---

# Issue 014: _⚠️ Potential issue_ | _🔴 Critical_
## Review Comment

_⚠️ Potential issue_ | _🔴 Critical_

**Persist delivery state only after queue admission succeeds.**

These paths update `latestSeq`, `currentContent`, `final`, and `errorText` before `enqueueEventLocked(...)` can reject the event. If the route is saturated, the broker returns an error with the in-memory snapshot already advanced even though nothing was queued. That is especially dangerous for terminal/error events because the delivery can become permanently "final" without any final message or resume signal ever reaching the adapter.



Also applies to: 411-429, 467-499, 808-880

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/channels/delivery_broker.go` around lines 319 - 340, The code
mutates delivery state (delivery.latestSeq, latestEventType, currentContent,
final, errorText, updatedAt) and calls recordDeliveryFailureLocked before
calling enqueueEventLocked, allowing enqueue failures to leave the in-memory
delivery advanced; change the flow so you prepare any values from normalized but
do NOT assign them to delivery until after enqueueEventLocked succeeds: call
route := b.ensureRouteLocked(...), err = b.enqueueEventLocked(route, delivery,
normalized) while still holding b.mu (or document lock handoff), and only on err
== nil then set delivery.latestSeq = normalized.Seq, delivery.latestEventType =
normalized.EventType, delivery.currentContent = normalized.Content,
delivery.final = normalized.Final, delivery.updatedAt = b.now(), and if
EventType == DeliveryEventTypeError set delivery.errorText =
deliveryErrorText(normalized.Metadata) and call
b.recordDeliveryFailureLocked(...) else clear delivery.errorText; then unlock
and call b.signalRoute(route). Ensure the same change is applied to the other
mentioned blocks (lines ~411-429, 467-499, 808-880) to avoid state advancement
on enqueue failure.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - `Deliver`, `ProjectEvent`, and `FailSession` currently advance in-memory delivery state before queue admission succeeds. If `enqueueEventLocked(...)` rejects the event, the snapshot can become newer/final/error even though nothing was queued.
  - The root fix is to compute the next event/state first, enqueue it, and only then commit the delivery-state mutations and failure metrics. I will add regression coverage around queue-saturation and failed admission paths.
  - Resolution: Reordered broker state commits after successful queue admission in [internal/channels/delivery_broker.go](/Users/pedronauck/Dev/projects/_worktrees/channels/internal/channels/delivery_broker.go:286) and added rejected-admission regressions in [internal/channels/delivery_broker_test.go](/Users/pedronauck/Dev/projects/_worktrees/channels/internal/channels/delivery_broker_test.go:428) and [internal/channels/delivery_projection_test.go](/Users/pedronauck/Dev/projects/_worktrees/channels/internal/channels/delivery_projection_test.go:183); verified with `make verify`.
