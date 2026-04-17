---
status: resolved
file: internal/api/core/more_coverage_test.go
line: 254
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4122628916,nitpick_hash:428ff9790e26
review_hash: 428ff9790e26
source_review_id: "4122628916"
source_review_submitted_at: "2026-04-16T16:31:31Z"
---

# Issue 014: Prefer a table-driven t.Run("Should...") matrix here.
## Review Comment

These branches differ mostly by fixture wiring and expected status, so the repetition makes the coverage harder to scan and extend than necessary. Folding them into a table-driven subtest matrix would keep the intent tighter and the failures more localized.

As per coding guidelines, "`**/*_test.go`: Use table-driven tests with subtests (`t.Run`) as default`" and "`MUST use `t.Run(\"Should...\")` pattern for ALL test cases`."

Also applies to: 493-544

## Triage

- Decision: `INVALID`
- Notes: This is a style/refactor request, not a correctness defect. The current tests already isolate distinct error branches with explicit names, and converting them to a table-driven matrix would only restructure test code without improving behavioral coverage for this batch.
