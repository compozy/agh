---
status: resolved
file: internal/extension/channel_delivery_notifier_test.go
line: 362
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4093721845,nitpick_hash:bb1212628ddf
review_hash: bb1212628ddf
source_review_id: "4093721845"
source_review_submitted_at: "2026-04-11T12:28:05Z"
---

# Issue 028: Consider using a channel-based wait instead of polling.
## Review Comment

The `waitForExtensionDeliveryCalls` uses `time.Sleep` polling. While acceptable in tests, a channel-based notification would be more responsive and eliminate arbitrary sleep durations.

This is a minor optimization since the current implementation works correctly for test purposes.

## Triage

- Decision: `valid`
- Why: `waitForExtensionDeliveryCalls` polls with `time.Sleep(10ms)`, which adds avoidable latency and leaves the test helper dependent on arbitrary sleep intervals.
- Root cause: The recording delivery transport has no wait/notification mechanism, so the helper resorts to periodic polling.
- Fix plan: Add a lightweight notification channel to the recording transport and have the wait helper block on notifications plus a real timeout.
- Resolution: Replaced the polling helper with notification-driven waiting and verified the extension package plus the repo gate.
