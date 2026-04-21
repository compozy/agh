---
status: resolved
file: internal/task/manager_test.go
line: 3089
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4135285786,nitpick_hash:17a1fd0456bf
review_hash: 17a1fd0456bf
source_review_id: "4135285786"
source_review_submitted_at: "2026-04-18T23:45:16Z"
---

# Issue 016: Use the repo’s required t.Run("Should...") shape for these new tests.
## Review Comment

Both additions are top-level single-case tests. Please wrap them in `t.Run("Should ...")` subtests, or fold them into a small table, so they match the default Go test structure used in this repository and stay easy to extend. As per coding guidelines "Use table-driven tests with subtests (`t.Run`) as default in Go tests" and "`**/*_test.go`: MUST use t.Run("Should...") pattern for ALL test cases".

## Triage

- Decision: `valid`
- Notes:
  - The new task-manager tests added in this area are single-case top-level test bodies.
  - This repository’s Go test convention expects test cases to be expressed as `t.Run("Should ...")` subtests, even when there is only one case today, to keep future extension consistent.
  - Fix approach: wrap the affected test bodies in `t.Run("Should ...")` subtests while preserving their current assertions.
  - Resolved by wrapping the affected task-manager tests in `Should ...` subtests while preserving and extending their assertions.
