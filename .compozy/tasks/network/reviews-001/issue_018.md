---
status: resolved
file: internal/network/helpers_test.go
line: 12
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4093722580,nitpick_hash:3b993654d0a2
review_hash: 3b993654d0a2
source_review_id: "4093722580"
source_review_submitted_at: "2026-04-11T12:29:15Z"
---

# Issue 018: Consider using t.Run for validation loop iterations.
## Review Comment

The loop validating each `Kind` would benefit from subtests for clearer failure reporting when a specific kind fails.

---

## Triage

- Decision: `valid`
- Root cause: the validation loop over enum kinds in `helpers_test.go` does not use subtests, so a single failure gives less precise output than the repo standard expects.
- Fix approach: convert the loop to named `t.Run("Should...")` subtests.
