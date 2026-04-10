---
status: resolved
file: internal/cli/extension_test.go
line: 79
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4092736828,nitpick_hash:80e5e35628c7
review_hash: 80e5e35628c7
source_review_id: "4092736828"
source_review_submitted_at: "2026-04-10T22:18:10Z"
---

# Issue 012: Unused variable deps after checksum mismatch test.
## Review Comment

Line 100 has `_ = deps` which serves no purpose and appears to be leftover from a refactor. The `deps` variable is created on line 82 but only `homePaths` is used in this test.

## Triage

- Decision: `valid`
- Notes: `deps` is unused in this test and `_ = deps` is a leftover compiler pacifier. I will remove the dead binding so the test states its real dependencies directly.
