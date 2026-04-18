---
status: resolved
file: internal/daemon/harness_context.go
line: 396
severity: minor
author: coderabbitai[bot]
provider_ref: review:4135189365,nitpick_hash:672c5f20b1c1
review_hash: 672c5f20b1c1
source_review_id: "4135189365"
source_review_submitted_at: "2026-04-18T22:38:10Z"
---

# Issue 012: Use acp.PromptTurnSourceSynthetic constant for consistency.
## Review Comment

Line 420 should set `meta.TurnSource` to `acp.PromptTurnSourceSynthetic` instead of `string(TurnOriginSynthetic)` to match the pattern established in lines 367-369, which correctly use the `acp` constants for user and network sources.

## Triage

- Decision: `valid`
- Notes:
  - The current code assigns `meta.TurnSource = string(TurnOriginSynthetic)`, which happens to work because both values are `"synthetic"`.
  - The daemon already uses ACP turn-source constants for user and network paths, so synthetic should use the same source of truth for consistency and future-proofing.
  - I will switch this assignment to `acp.PromptTurnSourceSynthetic`.
