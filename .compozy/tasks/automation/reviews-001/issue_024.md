---
status: resolved
file: internal/config/automation.go
line: 227
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4093724766,nitpick_hash:56af6e944c89
review_hash: 56af6e944c89
source_review_id: "4093724766"
source_review_submitted_at: "2026-04-11T12:31:10Z"
---

# Issue 024: Unused error return value.
## Review Comment

`toAutomationJob` always returns `nil` error but the signature includes an error return. If this is intentional for future extensibility, consider documenting it; otherwise, simplify the signature.

## Triage

- Decision: `valid`
- Notes: `parsedAutomationJob.toAutomationJob` never returns a non-nil error, so the extra return value and caller branching add dead complexity without carrying information. I will simplify the helper signature and update its call site accordingly.
