---
provider: coderabbit
pr: "120"
round: 1
round_created_at: 2026-05-07T18:15:52.56459Z
status: resolved
file: internal/bridgesdk/runtime_refac_test.go
line: 115
severity: minor
author: coderabbitai[bot]
provider_ref: review:4245882823,nitpick_hash:541bbd252baf
review_hash: 541bbd252baf
source_review_id: "4245882823"
source_review_submitted_at: "2026-05-07T16:38:59Z"
---

# Issue 025: Assert that the decoded object is empty, not just non-nil.
## Review Comment

This test is meant to lock down "`null` becomes `{}`". A non-nil map with unexpected keys would still pass, so add a `len(target) == 0` assertion.

As per coding guidelines, "Ensure tests verify behavior outcomes, not just function calls".

## Triage

- Decision: `valid`
- Root cause: the whitespace-`null` decode test only proves the destination map is non-nil; it does not prove that decoding produced an empty object rather than a map with unexpected keys.
- Fix plan: keep the non-nil assertion and add a `len(target) == 0` outcome check.
- Resolution: implemented and verified with focused Go tests, race-enabled package tests, and full `rtk make verify`.
