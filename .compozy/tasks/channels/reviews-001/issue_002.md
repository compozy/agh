---
status: resolved
file: internal/api/core/channels.go
line: 96
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4093721845,nitpick_hash:c59c4edf7a6d
review_hash: c59c4edf7a6d
source_review_id: "4093721845"
source_review_submitted_at: "2026-04-11T12:28:05Z"
---

# Issue 002: Missing whitespace normalization on path parameter.
## Review Comment

Same inconsistency as line 74—`c.Param("id")` should be trimmed for consistency with other handlers.

---

## Triage

- Decision: `invalid`
- Notes:
  - `UpdateChannelRequest.ToUpdateInstanceRequest(...)` already trims the supplied path id internally in [internal/api/contract/channels.go](/Users/pedronauck/Dev/projects/_worktrees/channels/internal/api/contract/channels.go:53).
  - Passing the raw `c.Param("id")` here does not leak whitespace into the domain request, so the reported bug does not reproduce on the current code path.
  - Resolution: Closed as invalid after code inspection; `make verify` passed without any required change for this finding.
