---
status: resolved
file: internal/api/udsapi/agent_channels_test.go
line: 190
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4177060832,nitpick_hash:898ff2d8ed55
review_hash: 898ff2d8ed55
source_review_id: "4177060832"
source_review_submitted_at: "2026-04-26T14:53:33Z"
---

# Issue 018: Verify WaitInbox is correctly invoked only for wait=true.
## Review Comment

The test validates `WaitInbox` is called with the correct parameters, but there's no explicit negative test ensuring `Inbox` (not `WaitInbox`) is called when `wait=false`. The `InboxFn` callback will `t.Fatal` if called, but only during the wait=true scenario.

Consider adding a separate subtest or case that explicitly verifies `wait=false` uses `Inbox` instead of `WaitInbox`.

---

## Triage

- Decision: `VALID`
- Notes: The current receive test proves `wait=true` calls `WaitInbox`, but it does not prove `wait=false` uses non-blocking `Inbox` and avoids `WaitInbox`. Fix by adding a separate subtest for the default path that fails if `WaitInbox` is invoked and asserts the response is built from `Inbox`.
