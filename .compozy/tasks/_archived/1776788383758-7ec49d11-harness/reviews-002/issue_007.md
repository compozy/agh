---
status: resolved
file: internal/daemon/harness_detached_work.go
line: 600
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4135285786,nitpick_hash:2dfdd59eddc4
review_hash: 2dfdd59eddc4
source_review_id: "4135285786"
source_review_submitted_at: "2026-04-18T23:45:16Z"
---

# Issue 007: Consider collision risk with truncated SHA-256 hash.
## Review Comment

The task ID uses only the first 8 bytes (16 hex chars) of a SHA-256 hash. While collisions are statistically unlikely for typical usage, this provides ~64 bits of collision resistance. For a safety-critical system with high task volume, you may want to use more bytes or document the collision probability assumptions.

## Triage

- Decision: `invalid`
- Notes:
  This is a theoretical collision-risk observation, not a concrete defect in the current detached-harness implementation. The ID is derived from `(ownerSessionID, submissionKey)` with a 64-bit truncated SHA-256 suffix, which is already well beyond the expected local daemon task volume for this feature. Expanding the identifier would change stable persisted IDs without an actual requirement or demonstrated collision scenario, so I am not changing it in this batch.
