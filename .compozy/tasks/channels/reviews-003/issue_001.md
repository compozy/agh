---
status: resolved
file: internal/api/udsapi/udsapi_integration_test.go
line: 319
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4093890324,nitpick_hash:05503626fe46
review_hash: 05503626fe46
source_review_id: "4093890324"
source_review_submitted_at: "2026-04-11T15:00:20Z"
---

# Issue 001: Optional DRY refactor: consolidate duplicated start/restart transition flow.
## Review Comment

`StartInstance` and `RestartInstance` share the same two-step transition logic. Extracting a helper would reduce duplication and keep error wording consistent.

## Triage

- Decision: `Invalid`
- Notes:
  This is a localized test-helper refactor, not a behavioral defect. The duplicated start/restart flow lives in both the UDS and HTTP integration helpers, so changing only this file would be partial stylistic churn without improving correctness or preventing a real regression.
  Closed with no code change after inspection confirmed the current duplication does not create inconsistent behavior in the scoped tests.
