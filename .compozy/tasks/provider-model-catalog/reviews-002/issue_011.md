---
provider: coderabbit
pr: "118"
round: 2
round_created_at: 2026-05-07T18:16:18.885242Z
status: resolved
file: internal/config/provider.go
line: 1152
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4245938208,nitpick_hash:b0289aab1b09
review_hash: b0289aab1b09
source_review_id: "4245938208"
source_review_submitted_at: "2026-05-07T16:46:43Z"
---

# Issue 011: Consider including the parse error in the validation message for debugging.
## Review Comment

When `time.ParseDuration` fails, the original error contains useful information about what was wrong with the input. Wrapping it with `%w` would help operators debug malformed duration strings.

## Triage

- Decision: `valid`
- Notes:
  - `validatePositiveDuration` in `internal/config/provider.go` drops the original `time.ParseDuration` error and always returns the generic message.
  - That makes malformed values harder to debug even though the parser already reports the exact problem.
  - Fix plan: preserve the human-facing validation prefix while wrapping the parse error with `%w`.
  - Fixed in `internal/config/provider.go` with regression coverage in `internal/config/provider_test.go`, then verified with focused package tests plus `make verify`.
