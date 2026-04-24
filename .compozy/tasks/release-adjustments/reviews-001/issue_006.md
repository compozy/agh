---
status: resolved
file: internal/api/httpapi/handlers_test.go
line: 1339
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4172207861,nitpick_hash:8478231e8896
review_hash: 8478231e8896
source_review_id: "4172207861"
source_review_submitted_at: "2026-04-24T17:07:23Z"
---

# Issue 006: Use a t.Run("Should...") scenario wrapper for this test case.
## Review Comment

The behavioral coverage is strong; please wrap this single scenario in the required subtest style for consistency.

As per coding guidelines "Use table-driven tests with subtests (`t.Run`) as default pattern for Go tests" and "MUST use t.Run(\"Should...\") pattern for ALL test cases".

## Triage

- Decision: `VALID`
- Notes:
  - `TestPromptSessionHandlerDrainsPromptAfterRequestCancellation` in `internal/api/httpapi/handlers_test.go` has a single scenario without the required `Should...` subtest wrapper.
  - The fix is to wrap the existing behavior in a named subtest. The test uses request cancellation and explicit goroutine synchronization, so the logic will remain unchanged.
