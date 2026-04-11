---
status: resolved
file: internal/api/spec/spec_test.go
line: 129
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4092845293,nitpick_hash:b836db48379b
review_hash: b836db48379b
source_review_id: "4092845293"
source_review_submitted_at: "2026-04-10T23:04:43Z"
---

# Issue 009: Assert enum set exactly, not only expected subset.
## Review Comment

`assertEnumValues` currently passes when unexpected enum values are added. That weakens contract protection. Validate exact membership (and fail on non-string enum entries) so schema drift is caught.

As per coding guidelines, "Verify tests can fail when business logic changes."

## Triage

- Decision: `valid`
- Notes:
  - The finding is accurate. `assertEnumValues` only proves that the expected strings are included; it does not fail if extra enum values are added or if a non-string enum entry appears.
  - Root cause: the helper performs subset matching instead of exact contract validation.
  - Fix approach: tighten `assertEnumValues` to reject non-string entries and compare exact enum membership so schema drift is caught by the tests.
  - Resolution: implemented in `internal/api/spec/spec_test.go` and verified with focused package tests plus `make verify`.
