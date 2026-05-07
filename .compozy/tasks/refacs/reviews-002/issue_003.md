---
provider: coderabbit
pr: "120"
round: 2
round_created_at: 2026-05-07T19:41:55.305082Z
status: resolved
file: internal/api/testutil/apitest_test.go
line: 69
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4247165327,nitpick_hash:15e558dc3c74
review_hash: 15e558dc3c74
source_review_id: "4247165327"
source_review_submitted_at: "2026-05-07T19:37:05Z"
---

# Issue 003: Convert the two request scenarios to a table-driven subtest.
## Review Comment

This block has multiple scenarios (`withBody`/`withoutBody`) in one flow; table-driven subtests would reduce duplication and make future cases easier to add.

As per coding guidelines, "Use table-driven test layout for Go tests with multiple scenarios."

## Triage

- Decision: `valid`
- Notes:
  - `internal/api/testutil/apitest_test.go:69-109` exercises two request scenarios in one linear test body with duplicated assertions.
  - This is a true test-shape issue under the repo's AGH test conventions, not a behavioral bug.
  - Fix plan: convert the request cases into a table-driven `t.Run("Should ...")` loop while preserving the existing assertions.
  - Resolved: the request helper coverage is now table-driven subtests with the original behavior/assertions preserved.
