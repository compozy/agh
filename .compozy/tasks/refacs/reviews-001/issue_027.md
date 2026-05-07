---
provider: coderabbit
pr: "120"
round: 1
round_created_at: 2026-05-07T18:15:52.56459Z
status: resolved
file: internal/bridgesdk/webhook_refac_test.go
line: 31
severity: minor
author: coderabbitai[bot]
provider_ref: review:4245882823,nitpick_hash:0883d3bc7b7a
review_hash: 0883d3bc7b7a
source_review_id: "4245882823"
source_review_submitted_at: "2026-05-07T16:38:59Z"
---

# Issue 027: Missing status code and response body assertions in the retry-after subtest.
## Review Comment

The test only validates the `Retry-After` header but doesn't assert `recorder.Code` or the response body, which leaves the 429 status path untested.

As per coding guidelines: "Always assert both HTTP status code AND response body in tests; status-code-only assertions are insufficient."

## Triage

- Decision: `valid`
- Root cause: the retry-after subtest only checks the `Retry-After` header, leaving the HTTP status and body for the `429 slow down` path unasserted.
- Fix plan: add status and response-body assertions alongside the header check.
- Resolution: implemented and verified with focused Go tests, race-enabled package tests, and full `rtk make verify`.
