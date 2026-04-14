---
status: resolved
file: internal/api/core/automation_test.go
line: 654
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4108760624,nitpick_hash:79e0eafc5809
review_hash: 79e0eafc5809
source_review_id: "4108760624"
source_review_submitted_at: "2026-04-14T20:02:29Z"
---

# Issue 004: Add an anti-aliasing assertion for patched Task.
## Review Comment

The new patch assertions verify values, but not clone semantics. A direct pointer assignment regression would still pass.

## Triage

- Decision: `valid`
- Notes:
  The patched job assertion validates copied values but not clone semantics, so a future pointer-alias regression would still pass. I will mutate the source task config after patching and assert the patched job stays unchanged.
  Resolution: Strengthened the patch test by giving the patched task an owner and mutating the source config after `applyJobPatch`; the assertions now prove the patched task was cloned rather than aliased.
