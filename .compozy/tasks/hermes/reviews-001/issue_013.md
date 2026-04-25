---
status: resolved
file: internal/diagnostics/redact.go
line: 29
severity: major
author: coderabbitai[bot]
provider_ref: review:4175534665,nitpick_hash:d34a702657ad
review_hash: d34a702657ad
source_review_id: "4175534665"
source_review_submitted_at: "2026-04-25T12:34:13Z"
---

# Issue 013: Non-positive budgets currently disable the bound entirely.
## Review Comment

When `maxBytes <= 0`, this returns the full redacted payload instead of a bounded result. That breaks the function contract and can still overflow storage limits if a caller passes `0` or a negative budget.

## Triage

- Decision: `VALID`
- Notes: `RedactAndBound` returns the full redacted payload when `maxBytes <= 0`, so a bad caller budget disables the storage bound. Change non-positive budgets to return an empty bounded result and add tests for zero/negative limits.
