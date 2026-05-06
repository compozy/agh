---
provider: coderabbit
pr: "106"
round: 1
round_created_at: 2026-05-06T04:12:39.763475Z
status: resolved
file: internal/bridges/task_notifier.go
line: 183
severity: major
author: coderabbitai[bot]
provider_ref: review:4233115469,nitpick_hash:2cdee8d9dce2
review_hash: 2cdee8d9dce2
source_review_id: "4233115469"
source_review_submitted_at: "2026-05-06T04:12:03Z"
---

# Issue 005: Page past ignored records or subscriptions can get stuck forever.
## Review Comment

`listTaskNotificationRecords` is capped by `eventLimit`, but the cursor only advances after a delivery. If the first page contains only non-terminal events, or only superseded mismatches before the real terminal event, every sweep re-reads the same prefix and never reaches the later final record. This needs paging/progress semantics that can move past records which are no longer candidates.

## Triage

- Decision: `valid`
- Notes:
  - `deliverSubscription` only advances the cursor after a delivery. If a full page contains only ignored/non-terminal records or only superseded mismatches, the cursor never moves and later terminal records stay unreachable.
  - Existing tests cover later valid events inside the same page, but not pagination starvation across `eventLimit` boundaries.
  - Planned fix: add safe progress semantics for scanned records that can never become deliverable and add a regression test for page starvation.
  - Resolved: the notifier now advances the durable cursor to the highest safely scanned sequence, including ignored and mismatched records, and regression tests cover the starvation case across pages.
