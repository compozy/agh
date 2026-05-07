---
provider: coderabbit
pr: "118"
round: 2
round_created_at: 2026-05-07T18:16:18.885242Z
status: resolved
file: internal/modelcatalog/types.go
line: 156
severity: minor
author: coderabbitai[bot]
provider_ref: review:4245938208,nitpick_hash:d0fa112aec18
review_hash: d0fa112aec18
source_review_id: "4245938208"
source_review_submitted_at: "2026-05-07T16:46:43Z"
---

# Issue 024: Use the declared enum types for state fields instead of raw strings.
## Review Comment

`Model.AvailabilityState` and `SourceStatus.RefreshState` are currently `string`, which bypasses the type guarantees provided by `AvailabilityState` and `RefreshState`.

Also applies to: 181-181

## Triage

- Decision: `valid`
- Notes:
  - `Model.AvailabilityState` and `SourceStatus.RefreshState` are still declared as raw `string` fields even though the package defines dedicated enum types for both domains.
  - That weakens compile-time guarantees inside the model-catalog/store code paths and makes invalid state assignments easier.
  - Fix plan: type the fields with `AvailabilityState` and `RefreshState`, then adjust store/boundary conversions where plain strings are still required.
