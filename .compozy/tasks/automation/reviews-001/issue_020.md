---
status: resolved
file: internal/cli/automation.go
line: 1016
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4093724766,nitpick_hash:63f3b0e366b0
review_hash: 63f3b0e366b0
source_review_id: "4093724766"
source_review_submitted_at: "2026-04-11T12:31:10Z"
---

# Issue 020: Potential unnecessary time.Now() call when now function is provided.
## Review Comment

The code calls `time.Now()` unconditionally before checking if a custom `now` function is provided. While functionally correct (it gets overwritten), this is slightly wasteful and could be simplified.

## Triage

- Decision: `valid`
- Notes: `parseAutomationOptionalTimeFlag` always calls `time.Now()` before checking whether a custom clock function was injected, which is unnecessary work and makes the control flow noisier. I will only call the default clock when no custom `now` function is provided.
