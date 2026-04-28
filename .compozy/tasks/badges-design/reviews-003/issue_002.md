---
status: pending
file: internal/api/core/handlers_test.go
line: 214
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4191605807,nitpick_hash:30e708e01a81
review_hash: 30e708e01a81
source_review_id: "4191605807"
source_review_submitted_at: "2026-04-28T18:57:12Z"
---

# Issue 002: Strengthen repair response assertions for future regressions.
## Review Comment

Consider also asserting `persisted` and action payload details, since this test already exercises dry-run behavior.

## Triage

- Decision: `UNREVIEWED`
- Notes:
