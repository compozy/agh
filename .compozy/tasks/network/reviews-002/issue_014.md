---
status: resolved
file: internal/network/manager.go
line: 643
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4093857291,nitpick_hash:24a588be4568
review_hash: 24a588be4568
source_review_id: "4093857291"
source_review_submitted_at: "2026-04-11T14:15:44Z"
---

# Issue 014: Potential TOCTOU race in acquireBroadcastSubscription.
## Review Comment

There's a time-of-check to time-of-use (TOCTOU) race between lines 647-652 and 665-673. After releasing the lock to subscribe (line 652), another goroutine could add the space to `m.spaces`, causing duplicate subscription handling. While this is handled by `cleanupDuplicateBroadcastSubscription`, the pattern could be simplified by holding the lock during the entire operation or using a sync.Map.

However, holding the lock during a potentially blocking network call (subscribe) is not ideal. The current approach with duplicate cleanup is a reasonable trade-off.

## Triage

- Decision: `invalid`
- Notes: The current implementation already handles the post-subscribe race by detecting the duplicate under the lock and unsubscribing the extra subscription via `cleanupDuplicateBroadcastSubscription`. Holding the mutex across `Subscribe` would risk blocking manager state on network I/O, so there is no unresolved correctness bug to fix here.
