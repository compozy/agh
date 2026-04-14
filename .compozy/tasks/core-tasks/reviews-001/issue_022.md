---
status: resolved
file: internal/extension/host_api_test.go
line: 1006
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4106878777,nitpick_hash:26351a3e8ad4
review_hash: 26351a3e8ad4
source_review_id: "4106878777"
source_review_submitted_at: "2026-04-14T14:46:54Z"
---

# Issue 022: Avoid wall-clock polling in this test.
## Review Comment

The `time.Sleep` loop makes this test timing-sensitive and prone to CI flakiness. Prefer a completion signal from the fake driver/session recorder, or another deterministic synchronization primitive, instead of polling for `done`.

As per coding guidelines "Never use `time.Sleep()` in orchestration — use proper synchronization primitives".

## Triage

- Decision: `VALID`
- Notes:
  The test currently polls session storage with a `time.Sleep(10 * time.Millisecond)` loop until a `done` event appears. That makes the assertion timing-sensitive and violates the workspace rule against sleep-based orchestration.
  The root cause is that the test is waiting for prompt completion indirectly even though the session manager already exposes a deterministic turn-finished callback path. I will switch the test to synchronize on that completion signal instead of wall-clock polling.
  Resolution: Replaced the polling loop with `session.Manager.SetTurnEndNotifier`, waited on the explicit completion signal, and then asserted the stored `done` event after prompt completion.
