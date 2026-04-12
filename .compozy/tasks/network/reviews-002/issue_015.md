---
status: resolved
file: internal/network/manager.go
line: 934
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4093857291,nitpick_hash:c234a61a8ed5
review_hash: c234a61a8ed5
source_review_id: "4093857291"
source_review_submitted_at: "2026-04-11T14:15:44Z"
---

# Issue 015: Minor: Redundant strings.TrimSpace call.
## Review Comment

Line 934 calls `strings.TrimSpace(compactJSON(raw))` but `compactJSON` already trims the result at line 953-955.

---

## Triage

- Decision: `valid`
- Root cause: `compactJSON` already trims its output, so the second `strings.TrimSpace(compactJSON(raw))` call is redundant.
- Fix plan: Compute the compacted value once and reuse it for the return tuple.
