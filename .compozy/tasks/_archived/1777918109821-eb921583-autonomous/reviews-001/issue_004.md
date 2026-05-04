---
status: resolved
file: internal/api/core/agent_channels.go
line: 392
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4177060832,nitpick_hash:5bf7f2fd2d92
review_hash: 5bf7f2fd2d92
source_review_id: "4177060832"
source_review_submitted_at: "2026-04-26T14:53:33Z"
---

# Issue 004: mergeCoordinationChannels: Unreachable nil check after make.
## Review Comment

Line 416 checks `if merged == nil` but `merged` is initialized with `make(...)` on line 411, so it can never be nil. The slice may be empty but not nil.

---

## Triage

- Decision: `VALID`
- Notes: `mergeCoordinationChannels` initializes `merged` with `make`, so the `if merged == nil` branch is unreachable. The function already returns a non-nil empty slice when there are no channels. Fix by removing the dead nil check.
