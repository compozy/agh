---
status: resolved
file: internal/channels/delivery_broker_test.go
line: 556
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4093857889,nitpick_hash:0ff558dc015a
review_hash: 0ff558dc015a
source_review_id: "4093857889"
source_review_submitted_at: "2026-04-11T14:16:28Z"
---

# Issue 004: Replace sleep-based polling with explicit synchronization.
## Review Comment

These helpers spin on `time.Sleep`, which makes the async broker tests timing-sensitive under CI load. Prefer signaling from the fake transport or another synchronization primitive instead of fixed 10ms polling loops. As per coding guidelines, "Never use time.Sleep() in orchestration — use proper synchronization primitives".

## Triage

- Decision: `Valid`
- Notes:
  `waitForCalls` and `waitForCondition` currently use blind `time.Sleep` polling in asynchronous broker tests. That makes progress dependent on scheduler timing instead of explicit synchronization. The fix is to replace those waits with transport-backed notifications and direct state checks so the tests wait on real transitions instead of fixed delays.
  Resolved in `internal/channels/delivery_broker_test.go` by replacing sleep polling with transport-driven waits and direct backlog assertions, then verified with `go test ./internal/channels -count=1` and `make verify`.
