---
status: resolved
file: internal/api/core/channels.go
line: 156
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4093721845,nitpick_hash:5f5332b12459
review_hash: 5f5332b12459
source_review_id: "4093721845"
source_review_submitted_at: "2026-04-11T12:28:05Z"
---

# Issue 003: Missing whitespace normalization on path parameter.
## Review Comment

Same inconsistency—`c.Param("id")` should be trimmed.

---

## Triage

- Decision: `invalid`
- Notes:
  - `ChannelTestDeliveryRequest.ToResolveDeliveryTargetRequest(...)` trims the supplied path id before validation in [internal/api/contract/channels.go](/Users/pedronauck/Dev/projects/_worktrees/channels/internal/api/contract/channels.go:91).
  - The whitespace normalization already happens in the contract layer, so this handler call site is not a live bug.
  - Resolution: Closed as invalid after code inspection; `make verify` passed without any required change for this finding.
