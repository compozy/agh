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

# Issue 011: Prefer a table-driven t.Run("Should...") matrix here.
## Review Comment

These branches differ mostly by fixture wiring and expected status, so the repetition makes the coverage harder to scan and extend than necessary. Folding them into a table-driven subtest matrix would keep the intent tighter and the failures more localized.

As per coding guidelines, "`**/*_test.go`: Use table-driven tests with subtests (`t.Run`) as default`" and "`MUST use `t.Run(\"Should...\")` pattern for ALL test cases`."

Also applies to: 493-544

## Triage

- Decision: `INVALID`
- Reason: The current [internal/api/core/more_coverage_test.go](/Users/pedronauck/Dev/compozy/_worktrees/site/internal/api/core/more_coverage_test.go#L251) no longer contains the repeated branch matrix cited in the review comment. The referenced secondary line range is also absent, so there is no remaining duplicated structure in scope to refactor.

## Resolution

- Analysis complete. No code change was required because the cited duplication is not present in the current file.
