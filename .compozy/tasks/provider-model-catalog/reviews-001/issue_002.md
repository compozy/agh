---
provider: coderabbit
pr: "118"
round: 1
round_created_at: 2026-05-07T16:19:53.268066Z
status: resolved
file: internal/acp/types.go
line: 486
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4245741930,nitpick_hash:3cf3f45fcebc
review_hash: 3cf3f45fcebc
source_review_id: "4245741930"
source_review_submitted_at: "2026-05-07T16:19:15Z"
---

# Issue 002: Public Caps field may still allow unsynchronized access.
## Review Comment

While `CapsSnapshot()`, `setCaps()`, and `setConfigOptions()` properly synchronize via `capsMu`, the `Caps` field on `AgentProcess` remains public. Direct access to `p.Caps` bypasses the mutex, potentially causing data races. Consider making `Caps` unexported to enforce access through the synchronized methods.

Also applies to: 598-624

## Triage

- Decision: `invalid`
- Notes:
  - The race concern is theoretical in the current change set, but there is no scoped production regression here.
  - Current production reads use `CapsSnapshot()` and the in-package writers already clone behind `capsMu`; the direct `proc.Caps` reads found in the tree are test-only snapshot assertions after startup.
  - Making `AgentProcess.Caps` unexported would require a broad API change across many files outside this review batch with no concrete failing behavior to fix here.
