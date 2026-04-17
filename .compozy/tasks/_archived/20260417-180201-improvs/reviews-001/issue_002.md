---
status: resolved
file: internal/automation/trigger_test.go
line: 93
severity: major
author: coderabbitai[bot]
provider_ref: review:4130502052,nitpick_hash:152cf6f5be0f
review_hash: 152cf6f5be0f
source_review_id: "4130502052"
source_review_submitted_at: "2026-04-17T16:38:53Z"
---

# Issue 002: Use t.Run("Should...") for this new test case.
## Review Comment

The assertion is good, but this new case should follow the required subtest pattern.

As per coding guidelines, "MUST use t.Run(\"Should...\") pattern for ALL test cases."

## Triage

- Decision: `VALID`
- Notes:
  The new static-prompt scenario is a standalone test body, while this repo's
  test contract requires named `t.Run("Should...")` cases for test scenarios.
  Plan: wrap the assertion flow in a `ShouldReturnTrimmedStaticPrompt` subtest.
