---
provider: coderabbit
pr: "118"
round: 1
round_created_at: 2026-05-07T16:19:53.268066Z
status: resolved
file: internal/api/core/session_workspace.go
line: 53
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4245741930,nitpick_hash:33ce4eaeedd9
review_hash: 33ce4eaeedd9
source_review_id: "4245741930"
source_review_submitted_at: "2026-05-07T16:19:15Z"
---

# Issue 005: Make runtime-override errors always carry session.ErrInvalidRuntimeOverride
## Review Comment

`statusForSessionError` now maps `session.ErrInvalidRuntimeOverride` to 400, but `prefixedRuntimeOverrideErr` does not guarantee that sentinel is attached. Wrapping it there makes status mapping robust and deterministic.

Also applies to: 156-157, 184-190

## Triage

- Decision: `invalid`
- Notes:
  - `prefixedRuntimeOverrideErr(...)` is currently only used with `session.ValidateReasoningEffort(...)`, which already wraps `session.ErrInvalidRuntimeOverride`.
  - Both helper paths preserve the sentinel today: the prefixed branch wraps with `%w`, and the empty-prefix branch returns the original sentinel-bearing error unchanged.
  - There is no current status-mapping bug to fix in the scoped code; this is a future-proofing suggestion rather than a defect in the present behavior.
