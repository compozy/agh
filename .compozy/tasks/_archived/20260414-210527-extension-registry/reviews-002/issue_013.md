---
status: resolved
file: internal/registry/multi_test.go
line: 491
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4107316563,nitpick_hash:1e63b75a6f56
review_hash: 1e63b75a6f56
source_review_id: "4107316563"
source_review_submitted_at: "2026-04-14T15:47:27Z"
---

# Issue 013: Subtests should use "Should..." naming pattern.
## Review Comment

The subtests here ("search respects canceled context", "info requires slug", etc.) are descriptive but don't follow the `t.Run("Should...")` pattern specified in coding guidelines.

As per coding guidelines, "MUST use t.Run("Should...") pattern for ALL test cases".

## Triage

- Decision: `valid`
- Notes:
  Marked completed (resolved).
