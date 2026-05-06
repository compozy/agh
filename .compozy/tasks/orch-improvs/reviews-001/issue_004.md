---
provider: coderabbit
pr: "106"
round: 1
round_created_at: 2026-05-06T04:12:39.763475Z
status: resolved
file: internal/api/spec/spec.go
line: 4469
severity: major
author: coderabbitai[bot]
provider_ref: review:4233115469,nitpick_hash:24191b22804b
review_hash: 24191b22804b
source_review_id: "4233115469"
source_review_submitted_at: "2026-05-06T04:12:03Z"
---

# Issue 004: Document the default review-policy enum value too.
## Review Comment

The schema migration in this PR defaults `tasks.review_policy` to `"none"`, but this enum only advertises the routed policies. Clients generated from the OpenAPI document will reject the persisted default, and the spec no longer round-trips existing task data correctly.

## Triage

- Decision: `valid`
- Notes:
  - `taskReviewPolicyValues()` still omits `task.ReviewPolicyNone` even though the task model and config defaults support `"none"`.
  - That makes the OpenAPI enum stricter than persisted/runtime data and can break generated clients on round-trips.
  - Planned fix: include `"none"` in the spec enum values so the contract matches runtime truth.
  - Resolved: the spec helper now includes `taskpkg.ReviewPolicyNone`, and generated OpenAPI/TypeScript artifacts were refreshed via codegen.
