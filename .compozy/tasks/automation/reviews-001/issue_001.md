---
status: resolved
file: internal/api/contract/contract_test.go
line: 249
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4093724766,nitpick_hash:1bb0273c0673
review_hash: 1bb0273c0673
source_review_id: "4093724766"
source_review_submitted_at: "2026-04-11T12:31:10Z"
---

# Issue 001: Cover zero-value PATCH fields in HasChanges() tests.
## Review Comment

The risky case for pointer-based update DTOs is an explicit zero value like `Enabled: ptr(false)`: it should still count as a change. Right now this only proves string-backed fields, so a regression in bool handling would still pass.

As per coding guidelines, "MUST test meaningful business logic, not trivial operations".

## Triage

- Decision: `valid`
- Notes: `UpdateJobRequest.HasChanges()` and `UpdateTriggerRequest.HasChanges()` correctly treat non-nil pointer fields as changes, including explicit zero values, but the current test only exercises string-backed fields. I will extend the contract test to cover `Enabled: ptr(false)` so pointer-bool regressions fail explicitly.
