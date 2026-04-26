---
status: resolved
file: internal/api/core/network_test.go
line: 930
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4176489704,nitpick_hash:196605105379
review_hash: "196605105379"
source_review_id: "4176489704"
source_review_submitted_at: "2026-04-26T03:49:14Z"
---

# Issue 001: Wrap these new cases in t.Run("Should...") subtests.
## Review Comment

These scenarios are added as top-level tests, which breaks the test shape the rest of this file already follows. As per coding guidelines, `**/*_test.go`: MUST use `t.Run("Should...")` pattern for ALL test cases.

Also applies to: 1040-1184, 1186-1285

## Triage

- Decision: `valid`
- Notes:
  - The three network route regressions added at lines 930, 1040, and 1186 are still standalone top-level tests in a file that otherwise uses the repo's required `t.Run("Should...")` structure for scenario cases.
  - Root cause: the new coverage was added directly as new test functions instead of folding the assertions into named subtests.
  - Fix plan: wrap each scenario body in a `t.Run("Should...")` subtest, move `t.Parallel()` into the subtest closure, and keep the existing assertions unchanged.
