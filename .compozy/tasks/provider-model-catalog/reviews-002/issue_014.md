---
provider: coderabbit
pr: "118"
round: 2
round_created_at: 2026-05-07T18:16:18.885242Z
status: resolved
file: internal/extension/manager_model_source_test.go
line: 72
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4245938208,nitpick_hash:6fc3e3b55bcd
review_hash: 6fc3e3b55bcd
source_review_id: "4245938208"
source_review_submitted_at: "2026-05-07T16:46:43Z"
---

# Issue 014: Assert the denied error explicitly.
## Review Comment

`err == nil` is too broad and can hide wrong failure paths. Assert the expected denied error text/sentinel.

As per coding guidelines "MUST have specific error assertions (ErrorContains, ErrorAs)".

## Triage

- Decision: `invalid`
- Notes:
  - `internal/extension/manager_model_source_test.go` already asserts a specific failure shape: `errors.Is(err, toolspkg.ErrToolUnavailable)` plus the expected `not granted service method "models/list"` text.
  - The test is not merely checking `err == nil`; it already distinguishes the intended denial path from unrelated failures.
  - No code change is needed.
  - Resolved as invalid after branch inspection and full verification.
