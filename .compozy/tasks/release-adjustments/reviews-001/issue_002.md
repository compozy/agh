---
status: resolved
file: internal/acp/client_test.go
line: 360
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4172207861,nitpick_hash:9d13db1d0d63
review_hash: 9d13db1d0d63
source_review_id: "4172207861"
source_review_submitted_at: "2026-04-24T17:07:23Z"
---

# Issue 002: Wrap this in the repo’s required Should... subtest pattern.
## Review Comment

The assertions look good, but this new test skips both `t.Run("Should...")` and `t.Parallel()` even though it is independent.

As per coding guidelines, "MUST use t.Run("Should...") pattern for ALL test cases", "Use table-driven tests with subtests (`t.Run`) as default pattern for Go tests", and "Add `t.Parallel()` for independent subtests in Go tests".

## Triage

- Decision: `VALID`
- Notes:
  - `TestPromptActivityReporterReportsWhilePromptIsInFlight` currently runs assertions directly in the top-level test body and does not use the required `t.Run("Should...")` subtest wrapper.
  - The scenario is independent, so the fix is to move the existing assertions into a named `Should...` subtest with `t.Parallel()`.
