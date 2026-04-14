---
status: resolved
file: internal/api/core/automation_test.go
line: 625
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4108760624,nitpick_hash:e441fb3a58c3
review_hash: e441fb3a58c3
source_review_id: "4108760624"
source_review_submitted_at: "2026-04-14T20:02:29Z"
---

# Issue 002: Split the new helper assertions into t.Run("Should ...") subtests.
## Review Comment

These added task-related checks are currently embedded in one large test, which reduces failure isolation and diverges from the repo’s test-case structure policy.

As per coding guidelines, `**/*_test.go`: `MUST use t.Run("Should...") pattern for ALL test cases`.

---

## Triage

- Decision: `valid`
- Notes:
  The helper coverage in `TestAutomationHelperFunctionsAndErrors` is currently one large block, which weakens failure isolation and violates the repo test-structure rule. I will split the task-related helper checks into focused `t.Run("Should ...")` subtests.
  Resolution: Split the helper coverage into focused `Should ...` subtests for validation wrapping, timestamp parsing, payload decoding, job mapping, job patch cloning, trigger mapping, and status-code mapping.
