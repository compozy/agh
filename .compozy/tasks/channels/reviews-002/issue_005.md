---
status: resolved
file: internal/channels/delivery_projection_test.go
line: 366
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4093857889,nitpick_hash:06a166580ccd
review_hash: 06a166580ccd
source_review_id: "4093857889"
source_review_submitted_at: "2026-04-11T14:16:28Z"
---

# Issue 005: Split this validation matrix into table-driven subtests.
## Review Comment

This single test mixes snapshot validation, request validation, normalization, and metadata helpers, so one failure hides the rest and makes the broken path harder to spot. A small table with `t.Run("Should...")` subtests would fit the rest of this suite much better. As per coding guidelines, "Use table-driven tests with subtests (`t.Run`) as default" and "MUST use t.Run("Should...") pattern for ALL test cases".

## Triage

- Decision: `Valid`
- Notes:
  `TestDeliveryValidationAndMetadataHelpers` mixes several unrelated behaviors into one assertion chain, so the first failure hides the rest and makes regressions harder to localize. The fix is to split the coverage into `t.Run("Should ...")` subtests while preserving the same validation, normalization, and metadata checks.
  Resolved in `internal/channels/delivery_projection_test.go` by splitting the coverage into focused `Should...` subtests, then verified with `go test ./internal/channels -count=1` and `make verify`.
