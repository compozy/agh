---
status: resolved
file: internal/extension/host_api_test.go
line: 436
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4092736828,nitpick_hash:6919e92e189a
review_hash: 6919e92e189a
source_review_id: "4092736828"
source_review_submitted_at: "2026-04-10T22:18:10Z"
---

# Issue 023: Loop variable capture is no longer needed in Go 1.22+.
## Review Comment

The `tt := tt` pattern on line 437 was required for loop variable capture in older Go versions but is no longer necessary since Go 1.22 changed loop variable semantics. This is harmless but can be removed for cleaner code.

## Triage

- Decision: `valid`
- Notes:
  Same rationale as issue 018: with Go 1.25 loop variables are scoped per iteration, so the extra rebinding in this test table is redundant.
  Fix approach: remove the unnecessary `tt := tt` rebinding in the affected loop.
