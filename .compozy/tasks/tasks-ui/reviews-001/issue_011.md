---
status: resolved
file: internal/api/httpapi/handlers_test.go
line: 167
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4133273307,nitpick_hash:64e3948d1893
review_hash: 64e3948d1893
source_review_id: "4133273307"
source_review_submitted_at: "2026-04-18T02:17:09Z"
---

# Issue 011: Prefer t.Run("Should...") subtests for each expected handler binding.
## Review Comment

This keeps failures scoped to a single route assertion and aligns with the test standards.

As per coding guidelines, "MUST use t.Run("Should...") pattern for ALL test cases".

## Triage

- Decision: `invalid`
- Notes: This is a preferred test-shape change rather than a missing route assertion or binding bug. The current map-driven route check already validates every handler binding in the batch and reports the failing route key directly. No behavioral defect was identified.
