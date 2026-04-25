---
status: resolved
file: internal/retry/retry_test.go
line: 10
severity: major
author: coderabbitai[bot]
provider_ref: review:4175534665,nitpick_hash:638cc5fc063d
review_hash: 638cc5fc063d
source_review_id: "4175534665"
source_review_submitted_at: "2026-04-25T12:34:13Z"
---

# Issue 025: Align this suite with the required Should... subtest pattern.
## Review Comment

The scenarios are strong, but the file should be organized as table-driven `t.Run("Should...")` subtests to match the repository’s mandatory test style.

As per coding guidelines, "`**/*_test.go`: Use table-driven tests with subtests (`t.Run`) as default pattern for Go tests" and "`MUST use t.Run(\"Should...\") pattern for ALL test cases`".

## Triage

- Decision: `valid`
- Root cause: `internal/retry/retry_test.go` uses standalone top-level test bodies and combines multiple scenarios inline instead of organizing cases under `t.Run("Should...")` subtests, which violates the repository test style called out in the review.
- Fix approach: restructure the retry tests into explicit `Should...` subtests, using table-driven cases where scenarios vary by input, without weakening the assertions.
