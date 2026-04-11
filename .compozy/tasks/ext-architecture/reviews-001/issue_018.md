---
status: resolved
file: internal/extension/capability_test.go
line: 96
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4092736828,nitpick_hash:24ca4f7f926f
review_hash: 24ca4f7f926f
source_review_id: "4092736828"
source_review_submitted_at: "2026-04-10T22:18:10Z"
---

# Issue 018: Loop variable capture is no longer needed in Go 1.22+.
## Review Comment

The `tt := tt` pattern at lines 96 and 143 was necessary in older Go versions to capture the loop variable correctly for parallel tests. Since Go 1.22, loop variables are scoped per-iteration, making this unnecessary.

Also applies to: 143-143

## Triage

- Decision: `valid`
- Notes:
  The project targets Go 1.25, so the per-iteration loop-variable semantics already make `tt := tt` redundant in these subtests. This is a low-risk cleanup in the reviewed file only.
  Fix approach: remove the unnecessary rebinding at both reported loops.
