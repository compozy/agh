---
provider: coderabbit
pr: "88"
round: 2
round_created_at: 2026-05-02T22:54:45.308545Z
status: resolved
file: internal/api/core/bridges_test.go
line: 341
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4215360648,nitpick_hash:62d8c36b3c80
review_hash: 62d8c36b3c80
source_review_id: "4215360648"
source_review_submitted_at: "2026-05-02T18:22:08Z"
---

# Issue 006: Assert the renamed secret_ref/kind fields on the list response.
## Review Comment

This now only checks `binding_name`, so the test would still pass if the handler dropped or mis-mapped the renamed secret-binding fields. It would be worth locking in `secret_ref`, `kind`, and the absence of `secret_value` here as well.

## Triage

- Decision: `valid`
- Notes:
  - The list-response test currently checks only `binding_name`, so it would miss regressions in the renamed `secret_ref` and `kind` fields or an accidental reintroduction of `secret_value`.
  - This was a real contract gap in the test, not a production bug; I strengthened `internal/api/core/bridges_test.go` to assert `binding_name`, `secret_ref`, `kind`, and the absence of `secret_value`.
  - Verification: `make verify` passed with the stricter list-response assertions.
