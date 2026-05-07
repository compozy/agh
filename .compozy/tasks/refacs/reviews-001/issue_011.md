---
provider: coderabbit
pr: "120"
round: 1
round_created_at: 2026-05-07T18:15:52.56459Z
status: resolved
file: internal/api/spec/spec.go
line: 4507
severity: minor
author: coderabbitai[bot]
provider_ref: review:4245882823,nitpick_hash:41fa954096ab
review_hash: 41fa954096ab
source_review_id: "4245882823"
source_review_submitted_at: "2026-05-07T16:38:59Z"
---

# Issue 011: Deep-copy nested container values in cloneSpecValue.
## Review Comment

At Line 4509, only the top-level `map[string]any` is copied. Nested maps/slices remain shared and can still mutate registry-backed data returned by `Operations()`.

## Triage

- Decision: `VALID`
- Notes:
  `cloneSpecValue` shallow-copies only the top-level map. Nested `map[string]any` and `[]any` values remain shared and can still mutate registry-backed operation metadata returned by `Operations()`. The clone needs to recurse through JSON-like containers.
