---
status: resolved
file: internal/testutil/e2e/runtime_harness_lifecycle_test.go
line: 25
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4129384275,nitpick_hash:e80f84c159cd
review_hash: e80f84c159cd
source_review_id: "4129384275"
source_review_submitted_at: "2026-04-17T13:54:50Z"
---

# Issue 032: Consider handling the JSON encoding error.
## Review Comment

The error from `json.NewEncoder(w).Encode()` is ignored. While this is unlikely to fail in test scenarios, handling it would align with the coding guideline to never ignore errors.

## Triage

- Decision: `valid`
- Notes:
  The lifecycle test server ignores the result of `json.NewEncoder(w).Encode`.
  Even in a test server, dropping the error violates the repo rule against
  ignored errors. The response helper should surface encoding failure instead of
  discarding it.

## Resolution

- The lifecycle test now handles JSON encoding failures through a shared
  response helper instead of ignoring the encoder result.
