---
provider: coderabbit
pr: "106"
round: 1
round_created_at: 2026-05-06T04:12:39.763475Z
status: resolved
file: internal/daemon/review_router.go
line: 232
severity: major
author: coderabbitai[bot]
provider_ref: review:4233115469,nitpick_hash:d9066d1a84a6
review_hash: d9066d1a84a6
source_review_id: "4233115469"
source_review_submitted_at: "2026-05-06T04:12:03Z"
---

# Issue 013: Avoid orphaning reviewer sessions when bind fails.
## Review Comment

`Create` happens before `BindRunReviewSession`. If the bind step conflicts or errors after the session is created, the new reviewer session stays alive with no review attached. Duplicate notifications and transient store errors will leak system reviewer sessions over time.

## Triage

- Decision: `valid`
- Notes:
  - `routeRunReview` still creates a new reviewer session before `BindRunReviewSession`, with no cleanup if binding conflicts or fails afterward.
  - That leaks orphaned system reviewer sessions on transient store conflicts and duplicate notifications.
  - Planned fix: add explicit cleanup for router-created sessions when bind fails and cover the failure path with a review-router test.
  - Resolved: router-created reviewer sessions are now stopped with failure cause/details when binding fails, and a regression test covers the cleanup path.
