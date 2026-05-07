---
provider: coderabbit
pr: "120"
round: 1
round_created_at: 2026-05-07T18:15:52.56459Z
status: resolved
file: internal/api/core/bridges_test.go
line: 546
severity: minor
author: coderabbitai[bot]
provider_ref: review:4245882823,nitpick_hash:8e6a53a4cfb5
review_hash: 8e6a53a4cfb5
source_review_id: "4245882823"
source_review_submitted_at: "2026-05-07T16:38:59Z"
---

# Issue 007: Assert the rejection body, not just the 400.
## Review Comment

These cases currently pass for any bad-request path, including unrelated decode failures. Check the error payload for the rejected field so the test fails only when the operational-state validation regresses.

As per coding guidelines, "Assert both HTTP status code AND response body in tests — status-code-only assertions are insufficient".

## Triage

- Decision: `VALID`
- Notes:
  The bad-request cases only assert `400`, so any unrelated failure path would satisfy the test. The response body already carries field-specific validation detail and should be asserted so the test proves operational-state rejection specifically.
