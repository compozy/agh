---
provider: coderabbit
pr: "106"
round: 1
round_created_at: 2026-05-06T04:12:39.763475Z
status: resolved
file: internal/daemon/review_router.go
line: 165
severity: major
author: coderabbitai[bot]
provider_ref: review:4233115469,nitpick_hash:db616a6b6de3
review_hash: db616a6b6de3
source_review_id: "4233115469"
source_review_submitted_at: "2026-05-06T04:12:03Z"
---

# Issue 012: Detach review routing from request cancellation.
## Review Comment

This callback performs daemon-owned work, but it keeps the caller's context all the way through reviewer selection, session creation, and diagnostic recording. A client disconnect or request timeout can cancel the review before it is bound, leaving it stuck in the requested state.

As per coding guidelines, "Detached execution lifetime — work that outlives an HTTP/UDS request must detach via `context.WithoutCancel(ctx)`, never tie execution lifetime to request lifetime" and "`context.WithoutCancel` does NOT preserve deadlines — re-attach a deadline if needed".

## Triage

- Decision: `valid`
- Notes:
  - `OnRunReviewRequested` still routes/binds directly on the caller context.
  - A canceled HTTP/UDS request can therefore cancel daemon-owned review routing before the binding is recorded, leaving review state stuck at `requested`.
  - Planned fix: detach routing/diagnostic work from request cancellation and add a cancellation regression test.
  - Resolved: review routing now detaches daemon-owned work from caller cancellation with a dedicated context helper, and review-router tests cover canceled-request behavior.
