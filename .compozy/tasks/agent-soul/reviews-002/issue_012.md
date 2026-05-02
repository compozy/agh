---
provider: coderabbit
pr: "88"
round: 2
round_created_at: 2026-05-02T22:54:45.308545Z
status: resolved
file: internal/api/spec/authored_context.go
line: 401
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4215360648,nitpick_hash:82f34ead1dab
review_hash: 82f34ead1dab
source_review_id: "4215360648"
source_review_submitted_at: "2026-05-02T18:22:08Z"
---

# Issue 012: Apply defensive copy for consistency with Tags handling.
## Review Comment

`append([]OperationSpec(nil), authoredContextOperationRegistry...)` creates only a shallow copy of the outer slice. While `buildOperation()` receives `OperationSpec` by value (protecting against mutations), the code inconsistently handles nested slices—Tags receives a defensive copy at line 3454 (`append([]string(nil), spec.Tags...)`), but Parameters and Responses do not. For defensive consistency and to prevent future misuse, apply the same pattern to all slice fields.

## Triage

- Decision: `invalid`
- Notes:
  - `authoredContextOperations()` is package-private and only consumed by spec generation via `append(ops, authoredContextOperations()...)`; current consumers treat the returned data as read-only.
  - `buildOperation` already copies `Tags` into the OpenAPI operation, and it iterates `Parameters` and `Responses` by value before building transport objects.
  - The suggestion is speculative consistency cleanup without a concrete mutation bug in the current code path, so I am not expanding scope for it in this review batch.
