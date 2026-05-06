---
provider: coderabbit
pr: "106"
round: 1
round_created_at: 2026-05-06T04:12:39.763475Z
status: resolved
file: internal/daemon/native_review_tools.go
line: 348
severity: major
author: coderabbitai[bot]
provider_ref: review:4233115469,nitpick_hash:a57799fce42c
review_hash: a57799fce42c
source_review_id: "4233115469"
source_review_submitted_at: "2026-05-06T04:12:03Z"
---

# Issue 011: Map unbound-review failures to denied, and keep the fallback redacted.
## Review Comment

`LookupRunReviewForSession()` misses satisfy `ErrRunReviewNotFound`, so this switch returns 404 before `isReviewBindingError()` can translate them to `ReasonSessionDenied`. The `default` branch then returns the original error unchanged, which bypasses the redaction already computed in `message`. Missing reviewer bindings should be denied, and unmapped task errors should still surface through a redacted tool error.

As per coding guidelines, "Raw `claim_token` (`agh_claim_*`) ... MUST NEVER appear in ... error payloads".

## Triage

- Decision: `valid`
- Notes:
  - `nativeReviewToolError` still matches `ErrRunReviewNotFound` as `not_found` before the binding-denial check, so unbound reviewer lookups do not map to `ReasonSessionDenied`.
  - The `default` branch returns the raw error, which bypasses the already-computed redacted message and can leak claim tokens.
  - Planned fix: reorder the mapping so binding failures become denied and wrap unmapped review errors in a redacted tool error envelope instead of returning the raw error.
  - Resolved: review binding errors now map to denied/session-denied before not-found handling, and the fallback path wraps backend failures in a redacted tool error instead of returning raw errors.
