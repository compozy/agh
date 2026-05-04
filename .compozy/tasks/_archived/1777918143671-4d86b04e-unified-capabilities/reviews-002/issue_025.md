---
status: resolved
file: internal/session/manager_clear_test.go
line: 12
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4148870373,nitpick_hash:489e830728d6
review_hash: 489e830728d6
source_review_id: "4148870373"
source_review_submitted_at: "2026-04-21T15:20:42Z"
---

# Issue 025: Prefer t.Run("Should...") subtests for the new clear-conversation coverage.
## Review Comment

These two scenarios exercise the same API with duplicated harness/prompt setup. Folding them into table-driven subtests would make the new file easier to extend with more clear/reset cases.

As per coding guidelines, "Use table-driven tests with subtests (`t.Run`) as default pattern for Go tests" and "`**/*_test.go`: MUST use t.Run("Should...") pattern for ALL test cases".

## Triage

- Decision: `valid`
- Notes:
  The new clear-conversation coverage uses separate scenario bodies with repeated harness/prompt setup instead of the repo-standard `Should...` subtest shape.
  I will fold the scoped scenarios into `t.Run("Should ...")` subtests while keeping the existing assertions intact.
  Fixed and verified with targeted package tests plus `make verify`.
