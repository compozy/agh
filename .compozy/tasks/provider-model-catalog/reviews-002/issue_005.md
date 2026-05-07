---
provider: coderabbit
pr: "118"
round: 2
round_created_at: 2026-05-07T18:16:18.885242Z
status: resolved
file: internal/api/core/conversions_parsers_test.go
line: 34
severity: minor
author: coderabbitai[bot]
provider_ref: review:4245938208,nitpick_hash:e337e6cd995a
review_hash: e337e6cd995a
source_review_id: "4245938208"
source_review_submitted_at: "2026-05-07T16:46:43Z"
---

# Issue 005: Exercise normalization, not just pass-through.
## Review Comment

The new assertions use already-trimmed `"gpt-test"` / `"high"` values, so they won't fail if `SessionPayloadFromInfo` stops trimming these fields. Seed whitespace here and keep the current expected assertions to cover the conversion behavior end-to-end.

## Triage

- Decision: `valid`
- Notes:
  - `internal/api/core/conversions_parsers_test.go` seeds `Model: "gpt-test"` and `ReasoningEffort: "high"` already normalized.
  - That means the test does not prove `SessionPayloadFromInfo` trims runtime override fields before projecting them.
  - Fix plan: seed whitespace-padded runtime override values and keep the same trimmed expectations.
  - Fixed in `internal/api/core/conversions_parsers_test.go` and verified with focused package tests plus `make verify`.
