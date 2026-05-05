---
status: resolved
file: internal/session/manager_test.go
line: 942
severity: minor
author: coderabbitai[bot]
provider_ref: review:4148870373,nitpick_hash:24daf4c9e185
review_hash: 24daf4c9e185
source_review_id: "4148870373"
source_review_submitted_at: "2026-04-21T15:20:42Z"
---

# Issue 028: Rename subtests to the required Should... pattern.
## Review Comment

Coverage is good, but the subtest labels should use the mandated naming convention.

As per coding guidelines "MUST use t.Run("Should...") pattern for ALL test cases".

## Triage

- Decision: `valid`
- Notes:
  The `TestCancelPrompt` subtests currently use descriptive labels, but not the required `Should...` naming convention enforced in this repository.
  I will rename the scoped subtests to `Should...` labels only.
  Fixed and verified with targeted package tests plus `make verify`.
