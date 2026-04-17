---
status: resolved
file: internal/workref/ref_test.go
line: 15
severity: major
author: coderabbitai[bot]
provider_ref: review:4130502052,nitpick_hash:c21e8633d431
review_hash: c21e8633d431
source_review_id: "4130502052"
source_review_submitted_at: "2026-04-17T16:38:53Z"
---

# Issue 020: Adopt the required t.Run("Should...") naming pattern.
## Review Comment

Case/suite names currently don’t follow the mandated `Should...` format for test cases.

As per coding guidelines, "MUST use t.Run(\"Should...\") pattern for ALL test cases".

Also applies to: 74-81

## Triage

- Decision: `VALID`
- Notes:
  The constructor suite and case names do not follow the required `Should...`
  naming pattern. Plan: rename the table entries and suite labels to `Should...`
  forms without changing the constructor coverage itself.
