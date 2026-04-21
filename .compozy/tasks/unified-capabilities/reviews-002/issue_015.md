---
status: resolved
file: internal/e2elane/lanes_test.go
line: 163
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4148870373,nitpick_hash:9566a96203ae
review_hash: 9566a96203ae
source_review_id: "4148870373"
source_review_submitted_at: "2026-04-21T15:20:42Z"
---

# Issue 015: Wrap this new case in t.Run("Should...") (table/subtest style).
## Review Comment

This test is valid functionally, but it bypasses the required subtest pattern used in this repo for Go tests. Please convert it to a table-driven/subtest form (even with one case) and use a `Should...` subtest name.

As per coding guidelines, `**/*_test.go`: "Use table-driven tests with subtests (`t.Run`) as default pattern for Go tests" and "MUST use t.Run("Should...") pattern for ALL test cases".

## Triage

- Decision: `valid`
- Root cause: the new `TestRuntimeLaneIncludesHarnessPackageCoverage` case is a single top-level assertion block that does not follow the repo's default subtest shape.
- Fix plan: wrap the existing assertions in a `Should...` subtest and keep the behavior unchanged.
- Resolution: implemented and verified through targeted Go tests and a clean `make verify` run.
