---
provider: coderabbit
pr: "118"
round: 2
round_created_at: 2026-05-07T18:16:18.885242Z
status: resolved
file: internal/extension/model_source.go
line: 220
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4245938208,nitpick_hash:f1a7a2f237ce
review_hash: f1a7a2f237ce
source_review_id: "4245938208"
source_review_submitted_at: "2026-05-07T16:46:43Z"
---

# Issue 015: Calling .UTC() on potentially zero ExpiresAt may produce misleading timestamps.
## Review Comment

When `row.ExpiresAt` is zero (i.e., the extension didn't provide an expiration), calling `.UTC()` on it returns `0001-01-01 00:00:00 +0000 UTC` rather than preserving zero-value semantics. Consider guarding this similar to the `RefreshedAt` handling at lines 208-211.

## Triage

- Decision: `invalid`
- Notes:
  - `time.Time{}.UTC()` preserves zero-time semantics: the value remains `IsZero()==true` and still represents the zero timestamp.
  - The current `row.ExpiresAt.UTC()` call therefore does not introduce a new non-zero expiration or break zero-value handling in this code path.
  - No code change is needed.
  - Resolved as invalid after branch inspection and full verification.
