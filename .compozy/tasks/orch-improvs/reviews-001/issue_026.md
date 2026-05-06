---
provider: coderabbit
pr: "106"
round: 1
round_created_at: 2026-05-06T04:12:39.763475Z
status: resolved
file: internal/store/globaldb/global_db_notification_cursor.go
line: 124
severity: minor
author: coderabbitai[bot]
provider_ref: review:4233115469,nitpick_hash:98bdcf2da6e6
review_hash: 98bdcf2da6e6
source_review_id: "4233115469"
source_review_submitted_at: "2026-05-06T04:12:03Z"
---

# Issue 026: Idempotent advances should still clear stale diagnostics.
## Review Comment

If `RecordCursorError` sets `last_error` on the current delivery, replaying that same confirmed delivery hits this fast path and returns without clearing the error or refreshing `updated_at`. The cursor can stay stuck in an error state until a newer sequence arrives.

## Triage

- Decision: `valid`
- Notes: `AdvanceCursor` treats an idempotent replay (`same sequence + same delivery id`) as an immediate success and returns the previously loaded cursor unchanged. If `RecordCursorError` had set `last_error`, that fast path leaves the cursor stuck with stale diagnostics and does not refresh `updated_at`. Fix by treating same-delivery confirmation as a lightweight refresh that clears `last_error` and updates the persisted timestamp.
- Resolution: Idempotent cursor replay now refreshes the stored cursor state, clears stale diagnostics, and the cursor-store test now asserts the corrected behavior.
