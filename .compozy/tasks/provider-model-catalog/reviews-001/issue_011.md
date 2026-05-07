---
provider: coderabbit
pr: "118"
round: 1
round_created_at: 2026-05-07T16:19:53.268066Z
status: resolved
file: internal/cli/provider_models.go
line: 118
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4245741930,nitpick_hash:2f97a5f3486e
review_hash: 2f97a5f3486e
source_review_id: "4245741930"
source_review_submitted_at: "2026-05-07T16:19:15Z"
---

# Issue 011: Consider extracting duplicate row builder.
## Review Comment

The text row builder (lines 118-128) and JSON row builder (lines 129-139) are identical. If they're always the same, a single function could be passed to `listBundle` twice, reducing duplication.

## Triage

- Decision: `invalid`
- Notes:
  - This is a maintainability suggestion, not a correctness issue in the current behavior.
  - The duplicated row builders in `internal/cli/provider_models.go` are identical but harmless, and extracting them does not close any functional gap in this review batch.
  - Per the scoped-fix workflow, I am avoiding unrelated refactors without a concrete defect.
