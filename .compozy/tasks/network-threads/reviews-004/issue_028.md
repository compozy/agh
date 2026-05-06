---
provider: coderabbit
pr: "105"
round: 4
round_created_at: 2026-05-06T00:08:12.899766Z
status: resolved
file: internal/network/manager_test.go
line: 312
severity: minor
author: coderabbitai[bot]
provider_ref: review:4232273319,nitpick_hash:363b5624fa0b
review_hash: 363b5624fa0b
source_review_id: "4232273319"
source_review_submitted_at: "2026-05-05T23:45:49Z"
---

# Issue 028: Lock the conversation-order counters on read.
## Review Comment

`publishCountAtWrite` and `promptCountAtWrite` are written under `s.mu`, but these assertions read them directly. If `WriteConversationMessage` ever runs on a worker goroutine, this helper will start failing under `-race` even when the ordering is correct. Add tiny getter methods that read those fields under the same mutex and use those from the assertions.

As per coding guidelines, "Run tests with `-race` flag and ensure `CGO_ENABLED=1` for race-sensitive packages."

Also applies to: 389-390, 1699-1743

## Triage

- Decision: `VALID`
- Root cause: `recordingConversationStore` protects `publishCountAtWrite` and `promptCountAtWrite` behind `s.mu` during writes, but the assertions read those fields directly. That bypasses the helper’s synchronization contract and can become race-prone as the call paths evolve.
- Fix approach: add small getter methods that read the counters under the mutex and use those accessors in the ordering assertions.
- Verification: fixed in scoped code and validated with fresh `make verify`.
