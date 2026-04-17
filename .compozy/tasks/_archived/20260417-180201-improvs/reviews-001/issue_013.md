---
status: resolved
file: internal/registry/installer_test.go
line: 554
severity: major
author: coderabbitai[bot]
provider_ref: review:4130502052,nitpick_hash:2e772c4dfda4
review_hash: 2e772c4dfda4
source_review_id: "4130502052"
source_review_submitted_at: "2026-04-17T16:38:53Z"
---

# Issue 013: Use required t.Run("Should...") subtest structure for this case.
## Review Comment

This new test is valid functionally, but it bypasses the required subtest naming/pattern. Please convert this into table-driven subtests using `t.Run("Should...")`.

As per coding guidelines, `**/*_test.go`: "MUST use t.Run("Should...") pattern for ALL test cases" and "Use table-driven tests with subtests (`t.Run`) as default".

## Triage

- Decision: `VALID`
- Notes:
  The checksum stability case is currently a standalone test flow instead of a
  named `t.Run("Should...")` scenario. Plan: move the assertions under a named
  subtest so the file follows the repo's required test structure.
