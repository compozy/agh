---
provider: coderabbit
pr: "88"
round: 1
round_created_at: 2026-05-02T18:22:40.559088Z
status: pending
file: internal/api/httpapi/httpapi_integration_test.go
line: 1830
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4215360648,nitpick_hash:20cb9d36a35b
review_hash: 20cb9d36a35b
source_review_id: "4215360648"
source_review_submitted_at: "2026-05-02T18:22:08Z"
---

# Issue 011: Don’t silently discard the new secretValue argument in the bridge test double.
## Review Comment

Line 1833 currently ignores the new parameter entirely; failing fast when it is non-nil prevents false-positive integration coverage.

## Triage

- Decision: `UNREVIEWED`
- Notes:
