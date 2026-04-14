---
status: resolved
file: internal/api/core/bridges_test.go
line: 199
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4103023844,nitpick_hash:8121817bf7bb
review_hash: 8121817bf7bb
source_review_id: "4103023844"
source_review_submitted_at: "2026-04-14T02:37:32Z"
---

# Issue 002: Prefer table-driven subtests for the new provider handler test.
## Review Comment

Please convert this new case to a `t.Run("Should...")` table-driven shape to match the test suite standard.

As per coding guidelines, `**/*_test.go`: "Use table-driven tests with subtests (`t.Run`) as default".

## Triage

- Decision: `valid`
- Root cause: the new provider handler test is a one-off test body instead of following the repository default of table-driven subtests with `t.Run("Should...")`.
- Fix approach: reshape the test into the standard subtest form without changing coverage.
