---
provider: coderabbit
pr: "120"
round: 1
round_created_at: 2026-05-07T18:15:52.56459Z
status: resolved
file: internal/bridgesdk/hostapi_refac_test.go
line: 12
severity: minor
author: coderabbitai[bot]
provider_ref: review:4245882823,nitpick_hash:d545ef054602
review_hash: d545ef054602
source_review_id: "4245882823"
source_review_submitted_at: "2026-05-07T16:38:59Z"
---

# Issue 024: Make the nil-context subtest assert the expected error.
## Review Comment

`err == nil` does not prove the nil-context guard fired; any unrelated failure would satisfy this test. Check the returned message or a sentinel error so the contract is actually locked down.

As per coding guidelines, "MUST have specific error assertions (ErrorContains, ErrorAs)".

## Triage

- Decision: `valid`
- Root cause: the nil-context Host API test only checks for a non-nil error, so any unrelated failure would pass without proving the `host api context is required` guard fired.
- Fix plan: assert the exact nil-context error while keeping the transport-call suppression check.
- Resolution: implemented and verified with focused Go tests, race-enabled package tests, and full `rtk make verify`.
