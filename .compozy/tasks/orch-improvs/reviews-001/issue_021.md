---
provider: coderabbit
pr: "106"
round: 1
round_created_at: 2026-05-06T04:12:39.763475Z
status: resolved
file: internal/situation/service_test.go
line: 1214
severity: minor
author: coderabbitai[bot]
provider_ref: review:4233115469,nitpick_hash:96a79075394b
review_hash: 96a79075394b
source_review_id: "4233115469"
source_review_submitted_at: "2026-05-06T04:12:03Z"
---

# Issue 021: Sort the stubbed reviews before applying Limit.
## Review Comment

Go map iteration is randomized, so slicing the last `Limit` entries from `s.reviews` makes any multi-review assertion non-deterministic. Sort the collected slice by a stable field before trimming it to keep these context-bundle tests reliable.

## Triage

- Decision: `valid`
- Notes: `taskStoreStub.ListRunReviews` iterates over `s.reviews`, which is a Go map, and then slices the collected results after filtering. That makes any `Limit` call nondeterministic in `internal/situation/service_test.go`, while the real GlobalDB implementation orders reviews by `updated_at DESC, review_id DESC`. Fix by sorting the stubbed slice to the same stable order before applying `Limit`.
- Resolution: Sorted the stubbed review slice by `UpdatedAt DESC, ReviewID DESC`, applied `Limit` after sorting, and added a targeted determinism test.
