---
provider: coderabbit
pr: "105"
round: 4
round_created_at: 2026-05-06T00:08:12.899766Z
status: resolved
file: internal/network/router_test.go
line: 1452
severity: minor
author: coderabbitai[bot]
provider_ref: review:4232273319,nitpick_hash:cb513c62f15e
review_hash: cb513c62f15e
source_review_id: "4232273319"
source_review_submitted_at: "2026-05-05T23:45:49Z"
---

# Issue 031: Make these validation checks fail for the reason the test names claim.
## Review Comment

Both assertions accept any non-nil error, and `OpenWork(Envelope{Kind: KindSay}, time.Time{})` is invalid in several ways before it ever proves the "non-opener" path. Use a fully valid non-opener envelope and assert the specific error so this test breaks on the intended regression.

As per coding guidelines, tests MUST have specific error assertions (ErrorContains, ErrorAs).

## Triage

- Decision: `VALID`
- Root cause: `TestWorkValidationErrors` accepts any non-nil error, and its `OpenWork` input is invalid for multiple reasons before it demonstrates the “non-opener” branch. That makes the test pass for the wrong failure mode.
- Fix approach: use a fully valid directed work-opening envelope with a non-opener kind and assert the specific sentinel error with `errors.Is`, so the test breaks only when the intended behavior regresses.
- Verification: fixed in scoped code and validated with fresh `make verify`.
