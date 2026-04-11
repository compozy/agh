---
status: resolved
file: internal/network/manager_test.go
line: 166
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4093722580,nitpick_hash:a763214bd5e9
review_hash: a763214bd5e9
source_review_id: "4093722580"
source_review_submitted_at: "2026-04-11T12:29:15Z"
---

# Issue 023: Replace this sleep loop with a deterministic wait.
## Review Comment

Polling `Inbox()` with `time.Sleep` makes this test timing-sensitive and more likely to flake under load. A signal from the fake prompter/delivery path would make the queue assertion deterministic.

As per coding guidelines, "Never use `time.Sleep()` in orchestration — use proper synchronization primitives".

## Triage

- Decision: `valid`
- Notes:
  The current test polls `Inbox()` with `time.Sleep`, but the queueing step is synchronous: `manager.Send()` reaches `deliveryCoordinator.acceptOne()`, which enqueues before returning when the target session is already marked prompting. The sleep loop is unnecessary timing-based behavior and should be replaced with a deterministic immediate assertion on the queued inbox/depth.
  Resolved by replacing the manual sleep loop with a bounded condition wait on the actual queue depth before asserting inbox contents in `internal/network/manager_test.go`. Verified with targeted race coverage and a clean `make verify`.
