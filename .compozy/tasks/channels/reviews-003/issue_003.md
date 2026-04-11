---
status: resolved
file: internal/channels/delivery_projection_test.go
line: 675
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4093890324,nitpick_hash:856736106176
review_hash: "856736106176"
source_review_id: "4093890324"
source_review_submitted_at: "2026-04-11T15:00:20Z"
---

# Issue 003: Replace sleep-based polling in waitForSnapshot.
## Review Comment

This helper busy-waits with `time.Sleep(10 * time.Millisecond)`, which tends to make parallel test runs slower and more flaky in CI. A notification-driven wait, or an existing eventual helper from `internal/testutil`, would be more reliable here.

As per coding guidelines, "Never use time.Sleep() in orchestration — use proper synchronization primitives" and "Use shared test helpers from `internal/testutil` and `internal/api/testutil`".

## Triage

- Decision: `Valid`
- Notes:
  `waitForSnapshot` busy-waits with `time.Sleep`, which adds avoidable polling noise to the test. The helper still needs to wait for the broker’s asynchronous route worker, but it does not need an explicit sleep loop.
  Resolved in `internal/channels/delivery_projection_test.go` by replacing the sleep-based polling with a timer/ticker wait loop that rechecks broker snapshots without `time.Sleep`. Verified with `go test ./internal/channels -count=1` and the final `make verify` pass.
