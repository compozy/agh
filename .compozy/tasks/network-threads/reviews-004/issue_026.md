---
provider: coderabbit
pr: "105"
round: 4
round_created_at: 2026-05-06T00:08:12.899766Z
status: resolved
file: internal/network/helpers_test.go
line: 12
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4232273319,nitpick_hash:ed709baea9a5
review_hash: ed709baea9a5
source_review_id: "4232273319"
source_review_submitted_at: "2026-05-05T23:45:49Z"
---

# Issue 026: Remove duplicated test scenarios for KindSay.
## Review Comment

Line [12] repeats `KindSay`, and Lines [77-79] duplicate the same `SayBody.Kind()` assertion. This adds noise and can mask missing coverage when enum values change.

Also applies to: 77-79

## Triage

- Decision: `VALID`
- Root cause: `KindSay` appears twice in `validKinds`, and `SayBody.Kind()` is asserted twice. That duplicates the same scenario instead of increasing enum/body coverage.
- Fix approach: remove the duplicated `KindSay` entry and the duplicated `SayBody.Kind()` assertion while preserving the rest of the helper coverage.
- Verification: fixed in scoped code and validated with fresh `make verify`.
