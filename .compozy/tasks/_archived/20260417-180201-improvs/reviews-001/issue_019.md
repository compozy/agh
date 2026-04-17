---
status: resolved
file: internal/transcript/transcript_test.go
line: 402
severity: major
author: coderabbitai[bot]
provider_ref: review:4130502052,nitpick_hash:1cd769796b09
review_hash: 1cd769796b09
source_review_id: "4130502052"
source_review_submitted_at: "2026-04-17T16:38:53Z"
---

# Issue 019: Use the required t.Run("Should...") pattern for this test case.
## Review Comment

Line 402 adds a standalone test body; this repo’s test policy requires subtests using `t.Run("Should...")` (table-driven by default).

As per coding guidelines, `**/*_test.go`: MUST use `t.Run("Should...")` pattern for ALL test cases and use table-driven tests with subtests (`t.Run`) as default.

## Triage

- Decision: `VALID`
- Notes:
  The raw-JSON tool-result case is written as a standalone test body instead of a
  named `t.Run("Should...")` scenario. Plan: keep the same assertions but move
  them under a `ShouldDecodeRawJSONObjectPayload` subtest.
