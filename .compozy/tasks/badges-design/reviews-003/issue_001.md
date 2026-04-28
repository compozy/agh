---
status: pending
file: internal/api/core/handlers.go
line: 422
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4191605807,nitpick_hash:10a98b0a97c6
review_hash: 10a98b0a97c6
source_review_id: "4191605807"
source_review_submitted_at: "2026-04-28T18:57:12Z"
---

# Issue 001: Consider rejecting conflicting dry_run alias values.
## Review Comment

If both `dry_run` and `dry-run` are sent with different values, current behavior silently prefers the first name.

## Triage

- Decision: `UNREVIEWED`
- Notes:
