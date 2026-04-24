---
status: resolved
file: internal/api/httpapi/handlers_error_test.go
line: 23
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4151198531,nitpick_hash:57271b62870a
review_hash: 57271b62870a
source_review_id: "4151198531"
source_review_submitted_at: "2026-04-21T23:03:23Z"
---

# Issue 002: Split this expanded error-mapping test into subtests for isolation.
## Review Comment

Now that this case covers five endpoints, one early failure hides the rest. Please convert to table-driven `t.Run("Should...")` subtests (and mark independent ones parallel) so each route/method mapping fails independently.

As per coding guidelines, "Use table-driven tests with subtests (`t.Run`) as default pattern for Go tests" and "MUST use t.Run("Should...") pattern for ALL test cases".

## Triage

- Decision: `valid`
- Notes:
  The HTTP handler error test currently checks create/get/resume/delete/stop in one linear flow, so the first failure hides later routes and it does not follow the repo's required subtest pattern. I will convert it to table-driven `t.Run("Should...")` subtests and mark independent subtests parallel.
