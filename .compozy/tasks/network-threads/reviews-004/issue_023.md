---
provider: coderabbit
pr: "105"
round: 4
round_created_at: 2026-05-06T00:08:12.899766Z
status: resolved
file: internal/hooks/network_dispatch_test.go
line: 23
severity: minor
author: coderabbitai[bot]
provider_ref: review:4232273319,nitpick_hash:79da9f7a6238
review_hash: 79da9f7a6238
source_review_id: "4232273319"
source_review_submitted_at: "2026-05-05T23:45:49Z"
---

# Issue 023: Use event-consistent fixtures for the direct-room case.
## Review Comment

Both the matcher and `networkDispatchTestPayload` force `Surface: "thread"` with a thread ID for every event. That means `HookNetworkDirectRoomOpened` never exercises the direct-surface / direct-ID path, so a regression there would still pass this test.

Also applies to: 84-103

## Triage

- Decision: `VALID`
- Root cause: `networkDispatchTestPayload` always builds a thread payload, and the hook declaration matcher also hard-codes `Surface:"thread"`. That means the `HookNetworkDirectRoomOpened` branch never exercises direct-surface matching or `direct_id` propagation.
- Fix approach: make the fixture event-aware so direct-room dispatch uses `surface=direct` plus `direct_id`, while thread events keep the thread-shaped payload. Keep the async dispatch assertions the same.
- Verification: fixed in scoped code and validated with fresh `make verify`.
