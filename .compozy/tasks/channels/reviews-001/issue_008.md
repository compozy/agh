---
status: resolved
file: internal/api/httpapi/channels_test.go
line: 13
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4093721845,nitpick_hash:f77ff4b10b06
review_hash: f77ff4b10b06
source_review_id: "4093721845"
source_review_submitted_at: "2026-04-11T12:28:05Z"
---

# Issue 008: Consolidate these handler tests into table-driven t.Run("Should...") cases.
## Review Comment

Line 13 through Line 131 repeats the same setup in three standalone tests; moving to table-driven subtests will reduce duplication and align with the repository test conventions.

As per coding guidelines, "Use table-driven tests with subtests (`t.Run`) as default" and "MUST use `t.Run("Should...")` pattern for ALL test cases".

## Triage

- Decision: `invalid`
- Notes:
  - These three tests cover distinct handler contracts with different request/response shapes (`create`, `routes`, and `test-delivery`) and already have focused assertions.
  - Converting them into one shared table would mostly add indirection without increasing regression detection, so I am keeping them explicit.
  - Resolution: Closed as invalid after code inspection; the focused tests remain unchanged and `make verify` passed.
