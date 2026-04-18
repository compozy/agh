---
status: resolved
file: internal/api/udsapi/helpers_test.go
line: 37
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4133264038,nitpick_hash:595f4de10fc3
review_hash: 595f4de10fc3
source_review_id: "4133264038"
source_review_submitted_at: "2026-04-18T02:14:16Z"
---

# Issue 010: Code duplication with internal/api/httpapi/helpers_test.go.
## Review Comment

The `stubSettingsService`, `stubSettingsRestartController`, `settingsTestSectionEnvelope`, and `settingsTestCollectionEnvelope` implementations are nearly identical to those in `internal/api/httpapi/helpers_test.go`. Consider extracting these to `internal/api/testutil` to follow the existing pattern of shared test helpers (like `StubSessionManager`, `StubObserver`, etc.) and reduce maintenance burden.

## Triage

- Decision: `invalid`
- Notes:
  This duplicates issue 006 and is likewise a maintainability suggestion rather than a defect. Extracting the stubs into `internal/api/testutil` would expand the write scope beyond the batch without fixing a broken behavior, so I am leaving the transport-local helpers in place.
