---
provider: coderabbit
pr: "90"
round: 1
round_created_at: 2026-05-03T03:31:47.363113Z
status: resolved
file: internal/subprocess/process.go
line: 278
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4215828964,nitpick_hash:d8bb2d262fea
review_hash: d8bb2d262fea
source_review_id: "4215828964"
source_review_submitted_at: "2026-05-03T03:31:19Z"
---

# Issue 008: Consider consolidating duplicate cleanup implementations.
## Review Comment

This function is nearly identical to `cleanupStartedTerminalCommand` in `internal/acp/handlers.go`. Both follow the same kill → wait → force-exit pattern. Consider extracting a shared helper in `procutil` to reduce duplication.

## Triage

- Decision: `invalid`
- Reasoning: the comment proposes a deduplication refactor between `internal/subprocess` and `internal/acp`, but it does not identify a current behavioral bug or security gap in `cleanupStartedManagedCommand`.
- Scope note: implementing it would require editing additional out-of-batch files (`internal/acp/handlers.go` and/or `internal/procutil`) solely for consolidation. That is outside the scoped remediation work.
