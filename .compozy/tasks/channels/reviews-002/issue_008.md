---
status: resolved
file: internal/daemon/daemon_test.go
line: 3126
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4093857889,nitpick_hash:b137a89c9c42
review_hash: b137a89c9c42
source_review_id: "4093857889"
source_review_submitted_at: "2026-04-11T14:16:28Z"
---

# Issue 008: Fire-and-forget goroutine may cause race with shutdown.
## Review Comment

The goroutines spawned by `sendDelayedDeliveryResult` are not tracked. When `releaseSlowDeliveries` is called during shutdown, pending goroutines unblock but there's no wait for them to complete sending results before the shutdown response is sent and the process potentially exits.

This could cause test flakiness if tests expect all delivery acks to arrive before the shutdown ack.

As per coding guidelines, "`**/*.go`: No fire-and-forget goroutines — track with sync.WaitGroup or equivalent`".

## Triage

- Decision: `Valid`
- Notes:
  The delayed delivery helper launches untracked goroutines that can still be encoding delivery acks after shutdown is released. Because shutdown currently acknowledges immediately after `releaseSlowDeliveries()`, tests can observe shutdown before those delayed results flush. The fix is to track delayed delivery senders with a `sync.WaitGroup` and wait for them after release before sending the shutdown acknowledgement.
  Resolved in `internal/daemon/daemon_test.go` by tracking delayed senders with a `WaitGroup` and waiting during shutdown, then verified with `go test ./internal/daemon -count=1` and `make verify`.
