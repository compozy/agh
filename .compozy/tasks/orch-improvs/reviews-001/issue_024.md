---
provider: coderabbit
pr: "106"
round: 1
round_created_at: 2026-05-06T04:12:39.763475Z
status: resolved
file: internal/situation/task_context.go
line: 413
severity: major
author: coderabbitai[bot]
provider_ref: review:4233115469,nitpick_hash:9e24227fae92
review_hash: 9e24227fae92
source_review_id: "4233115469"
source_review_submitted_at: "2026-05-06T04:12:03Z"
---

# Issue 024: Don’t return an over-budget bundle as success.
## Review Comment

The `default` branch exits even when `taskContextOverBudget` is still true. If the remaining fields alone exceed `maxBytes`, callers still get an oversized bundle and the configured context cap is no longer enforced.

## Triage

- Decision: `valid`
- Notes: `enforceTaskContextBudget` repeatedly trims optional sections, but the `default` branch currently returns the bundle as success even when the `taskContextOverBudget` check that entered the loop is still true. That means a task with an oversized non-trimmable core can silently exceed `ContextBodyMaxBytes`. Fix by returning a bounded error once no further trimming is possible instead of returning an oversized bundle.
- Resolution: The no-more-trimming path now returns `task.ErrPayloadTooLarge`, and a regression test covers the oversized untrimmable bundle case.
