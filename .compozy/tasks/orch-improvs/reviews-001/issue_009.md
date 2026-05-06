---
provider: coderabbit
pr: "106"
round: 1
round_created_at: 2026-05-06T04:12:39.763475Z
status: resolved
file: internal/cli/task.go
line: 824
severity: minor
author: coderabbitai[bot]
provider_ref: review:4233115469,nitpick_hash:3535b363925f
review_hash: 3535b363925f
source_review_id: "4233115469"
source_review_submitted_at: "2026-05-06T04:12:03Z"
---

# Issue 009: Reject negative review rounds and attempts in the CLI.
## Review Comment

`buildTaskRunReviewRequest` currently passes negative `--round` and `--attempt` values through unchanged. That turns obvious caller mistakes into server-side validation failures instead of immediate CLI feedback.

## Triage

- Decision: `valid`
- Notes:
  - `buildTaskRunReviewRequest` still forwards negative `round` and `attempt` values unchanged.
  - That converts deterministic caller input errors into server-side validation failures.
  - Planned fix: fail fast in the CLI builder for negative review round/attempt values and add regression coverage.
  - Resolved: the review-request builder now rejects negative round and attempt values, and CLI tests cover both invalid flags.
