---
provider: coderabbit
pr: "120"
round: 1
round_created_at: 2026-05-07T18:15:52.56459Z
status: resolved
file: internal/bridgesdk/errors_refac_test.go
line: 11
severity: minor
author: coderabbitai[bot]
provider_ref: review:4245882823,nitpick_hash:6a1b0807c564
review_hash: 6a1b0807c564
source_review_id: "4245882823"
source_review_submitted_at: "2026-05-07T16:38:59Z"
---

# Issue 023: Assert the concrete precondition error in both subtests.
## Review Comment

Right now both cases only check `err != nil`, so they'll still pass on any unrelated failure. These are contract tests for new guard clauses; assert the expected message or a sentinel error.

As per coding guidelines, "MUST have specific error assertions (ErrorContains, ErrorAs)".

## Triage

- Decision: `valid`
- Root cause: the new `RetryDo` guard-clause tests only assert `err != nil`, so unrelated failures would satisfy them and the contract for the nil-context / nil-operation preconditions would remain unpinned.
- Fix plan: assert the exact guard-clause messages and extend the refac coverage to keep the operation callback from being invoked on invalid inputs.
- Resolution: implemented and verified with focused Go tests, race-enabled package tests, and full `rtk make verify`.
