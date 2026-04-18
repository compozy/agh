---
status: resolved
file: internal/api/httpapi/helpers_test.go
line: 492
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4133264038,nitpick_hash:a27fb0131c13
review_hash: a27fb0131c13
source_review_id: "4133264038"
source_review_submitted_at: "2026-04-18T02:14:16Z"
---

# Issue 006: Duplicate helper functions with internal/api/udsapi/helpers_test.go.
## Review Comment

`settingsTestSectionEnvelope` and `settingsTestCollectionEnvelope` are duplicated across both test packages. Consider extracting to `internal/api/testutil` alongside the shared stubs.

## Triage

- Decision: `invalid`
- Notes:
  This is a refactor suggestion, not a correctness defect in the scoped batch. Extracting the duplicated envelopes would require touching `internal/api/testutil` and both transport packages purely for reuse, without changing behavior or reducing a concrete bug risk. I am leaving the helpers local to keep the transport tests isolated and the write scope constrained.
