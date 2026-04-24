---
status: resolved
file: internal/api/udsapi/handlers_error_test.go
line: 19
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4151198531,nitpick_hash:e290b3d34bb8
review_hash: e290b3d34bb8
source_review_id: "4151198531"
source_review_submitted_at: "2026-04-21T23:03:23Z"
---

# Issue 005: Refactor this endpoint-error matrix into t.Run("Should...") subtests.
## Review Comment

This block now covers multiple behaviors (create/get/resume/delete/stop) in one flow; converting it to table-driven subtests will improve failure locality and align with repo test conventions.

As per coding guidelines, `**/*_test.go`: MUST use t.Run("Should...") pattern for ALL test cases.

## Triage

- Decision: `valid`
- Notes:
  The UDS handler error test has the same monolithic structure as the HTTP version: multiple route checks are coupled in one flow, which hurts failure isolation and skips the required `Should...` subtest convention. I will convert it to table-driven parallel subtests.
