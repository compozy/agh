---
status: resolved
file: internal/network/delivery_test.go
line: 499
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4093722580,nitpick_hash:d79132174670
review_hash: d79132174670
source_review_id: "4093722580"
source_review_submitted_at: "2026-04-11T12:29:15Z"
---

# Issue 016: Avoid polling with time.Sleep in the concurrency helper.
## Review Comment

`waitForCalls` is coordinating goroutine progress, so the fixed 10ms sleep loop makes this suite timing-sensitive under load. Have `PromptNetwork` signal a channel or `sync.Cond` and wait on that here instead.

As per coding guidelines, "Never use time.Sleep() in orchestration — use proper synchronization primitives".

## Triage

- Decision: `valid`
- Root cause: `fakeDeliveryPrompter.waitForCalls` polls with `time.Sleep`, which makes the concurrency tests timing-sensitive under load.
- Fix approach: replace the polling loop with condition-based notification from `PromptNetwork`.
