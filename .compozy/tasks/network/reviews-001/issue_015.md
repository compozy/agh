---
status: resolved
file: internal/network/delivery_integration_test.go
line: 337
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4093722580,nitpick_hash:c3503f976a1d
review_hash: c3503f976a1d
source_review_id: "4093722580"
source_review_submitted_at: "2026-04-11T12:29:15Z"
---

# Issue 015: These polling helpers are still wall-clock dependent.
## Review Comment

The `time.Sleep` loops make the integration suite slower and prone to flakes on loaded runners. Prefer signaling from `integrationPromptDriver` / `deliveryCoordinator` with channels or a `sync.Cond` instead of retrying on a timer.

As per coding guidelines, "Never use `time.Sleep()` in orchestration — use proper synchronization primitives".

## Triage

- Decision: `valid`
- Root cause: the integration helpers wait for prompt and queue state by polling with `time.Sleep`, which is slower and more timing-sensitive than signaling on actual state changes.
- Fix approach: teach the integration prompt driver/helpers to signal prompt-count and delivery-progress changes through synchronization primitives instead of wall-clock polling.
