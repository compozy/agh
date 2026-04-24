---
status: resolved
file: internal/api/httpapi/handlers_test.go
line: 1074
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4151198531,nitpick_hash:2bfe6ce257e0
review_hash: 2bfe6ce257e0
source_review_id: "4151198531"
source_review_submitted_at: "2026-04-21T23:03:23Z"
---

# Issue 003: Use t.Run("Should...") for the new delete-session test case.
## Review Comment

Please wrap this new case in a `Should...` subtest to match the required test pattern used by repo guidelines.

As per coding guidelines, "MUST use t.Run("Should...") pattern for ALL test cases".

## Triage

- Decision: `valid`
- Notes:
  `TestDeleteSessionHandlerReturnsNoContent` is still a single top-level assertion block and does not follow the repository's required `t.Run("Should...")` pattern. I will wrap the delete-session assertions in a `Should...` subtest and make the test parallel-safe.
