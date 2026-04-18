---
status: resolved
file: internal/api/udsapi/handlers_test.go
line: 218
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4133273307,nitpick_hash:f90d63b6243c
review_hash: f90d63b6243c
source_review_id: "4133273307"
source_review_submitted_at: "2026-04-18T02:17:09Z"
---

# Issue 013: Use explicit t.Run("Should...") subtests for each expected route binding.
## Review Comment

This map-driven assertion block is a good table candidate; subtests will make failures isolated and guideline-compliant.

As per coding guidelines, "MUST use t.Run("Should...") pattern for ALL test cases".

## Triage

- Decision: `invalid`
- Notes: This is a style recommendation only. The current UDS route-binding test already checks the exact set of task handlers and produces route-specific failures. I did not find a correctness gap that requires restructuring it into explicit subtests.
