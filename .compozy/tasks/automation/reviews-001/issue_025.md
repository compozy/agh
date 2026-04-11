---
status: resolved
file: internal/config/automation.go
line: 255
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4093724766,nitpick_hash:e30e30d3f952
review_hash: e30e30d3f952
source_review_id: "4093724766"
source_review_submitted_at: "2026-04-11T12:31:10Z"
---

# Issue 025: Unused error return value.
## Review Comment

`toAutomationTrigger` always returns `nil` error but the signature includes an error return. Same consideration as `toAutomationJob` above.

---

## Triage

- Decision: `valid`
- Notes: `parsedAutomationTrigger.toAutomationTrigger` has the same unused error-return pattern as the job helper, with no current failure path. I will simplify the signature and its callers to match the actual behavior.
