---
provider: coderabbit
pr: "88"
round: 1
round_created_at: 2026-05-02T18:22:40.559088Z
status: pending
file: internal/api/core/bridges_test.go
line: 341
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4215360648,nitpick_hash:62d8c36b3c80
review_hash: 62d8c36b3c80
source_review_id: "4215360648"
source_review_submitted_at: "2026-05-02T18:22:08Z"
---

# Issue 007: Assert the renamed secret_ref/kind fields on the list response.
## Review Comment

This now only checks `binding_name`, so the test would still pass if the handler dropped or mis-mapped the renamed secret-binding fields. It would be worth locking in `secret_ref`, `kind`, and the absence of `secret_value` here as well.

## Triage

- Decision: `UNREVIEWED`
- Notes:
